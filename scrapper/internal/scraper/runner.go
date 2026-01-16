package scraper

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"scrapper/internal/aiclient"
	"scrapper/internal/models"
	"scrapper/internal/store"
)

const (
	// MaxConsecutiveErrors is the maximum number of consecutive page fetch errors before stopping
	MaxConsecutiveErrors = 10
)

// Runner handles background scraping with telemetry.
type Runner struct {
	store    *store.Store
	client   *Client
	aiClient *aiclient.Client // Optional AI processing client
}

// NewRunner creates a new Runner instance.
func NewRunner(s *store.Store, clientCfg ClientConfig) *Runner {
	return &Runner{
		store:  s,
		client: NewClient(clientCfg),
	}
}

// SetAIClient sets the AI processing client (optional).
func (r *Runner) SetAIClient(client *aiclient.Client) {
	r.aiClient = client
}

// RunResult contains the results of a scrape run.
type RunResult struct {
	RunID           int64
	Status          string
	JobsProcessed   int
	JobsInserted    int
	JobsUpdated     int
	JobsSkipped     int
	PagesScraped    int
	AIJobsProcessed int // Jobs sent to AI service
	AIJobsSkipped   int // Jobs skipped by AI service (already processed)
	AIJobsFailed    int // Jobs that failed AI processing
	Errors          []string
	StopReason      string
}

// Run performs background scraping with run tracking and incremental logic.
func (r *Runner) Run(ctx context.Context, req models.ScrapeRequest, runID int64) RunResult {
	result := RunResult{
		RunID:  runID,
		Status: "completed",
	}

	var (
		consecutiveErrors = 0
		stopScraping      = false
	)

	// Ensure run is updated when function exits
	defer func() {
		errLog := strings.Join(result.Errors, "\n")
		if len(result.Errors) > 0 && result.JobsInserted == 0 && result.JobsUpdated == 0 {
			result.Status = "failed"
		}

		if err := r.store.UpdateRun(ctx, runID, result.Status, result.JobsProcessed, result.JobsInserted, result.JobsUpdated, result.JobsSkipped, result.PagesScraped, errLog); err != nil {
			slog.Error("failed to update run", "run_id", runID, "error", err)
		}

		slog.Info("scraper completed",
			"run_id", runID,
			"status", result.Status,
			"strategy", req.Strategy,
			"jobs_processed", result.JobsProcessed,
			"jobs_inserted", result.JobsInserted,
			"jobs_updated", result.JobsUpdated,
			"jobs_skipped", result.JobsSkipped,
			"pages_scraped", result.PagesScraped,
			"ai_jobs_processed", result.AIJobsProcessed,
			"ai_jobs_skipped", result.AIJobsSkipped,
			"ai_jobs_failed", result.AIJobsFailed,
			"errors", len(result.Errors),
			"stop_reason", result.StopReason,
		)
	}()

	slog.Info("scraper started",
		"run_id", runID,
		"strategy", req.Strategy,
		"max_pages", req.MaxPages,
		"polite", req.Polite,
		"keywords", req.Keywords,
		"cantons", req.Cantons,
		"ai_enabled", r.aiClient != nil,
	)

pageLoop:
	for page := req.StartPage; req.MaxPages == 0 || page < req.StartPage+req.MaxPages; page++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			result.Status = "cancelled"
			result.StopReason = "context cancelled"
			break pageLoop
		default:
		}

		if stopScraping {
			break
		}

		slog.Info("fetching page", "run_id", runID, "page", page)

		jobs, err := r.client.FetchJobsWithRequest(&req, page)
		if err != nil {
			errMsg := "page " + strconv.Itoa(page) + ": " + err.Error()
			result.Errors = append(result.Errors, errMsg)
			slog.Error("failed to fetch jobs page", "run_id", runID, "page", page, "error", err)

			// Check for 412 error (API limit reached)
			if strings.Contains(err.Error(), "status 412") || strings.Contains(err.Error(), "exceed max result limit") {
				slog.Warn("API limit reached (412), stopping scrape", "run_id", runID, "page", page)
				result.StopReason = "API limit reached (412)"
				break pageLoop
			}

			// Track consecutive errors
			consecutiveErrors++
			if consecutiveErrors >= MaxConsecutiveErrors {
				slog.Warn("too many consecutive errors, stopping scrape",
					"run_id", runID,
					"consecutive_errors", consecutiveErrors,
				)
				result.StopReason = "too many consecutive errors"
				result.Status = "failed"
				break pageLoop
			}
			continue
		}

		// Reset consecutive error counter on success
		consecutiveErrors = 0
		result.PagesScraped++

		// If no jobs returned, we've reached the end
		if len(jobs) == 0 {
			slog.Info("no more jobs found, stopping", "run_id", runID, "page", page)
			result.StopReason = "no more jobs"
			break
		}

		slog.Info("fetched jobs from page", "run_id", runID, "page", page, "count", len(jobs))

		for _, job := range jobs {
			result.JobsProcessed++

			// Check if job exists and get its last updated time
			dbUpdatedTime, found, err := r.store.GetJobLastUpdated(ctx, job.ID)
			if err != nil {
				errMsg := "check " + job.ID + ": " + err.Error()
				result.Errors = append(result.Errors, errMsg)
				slog.Error("failed to check job", "run_id", runID, "id", job.ID, "error", err)
				continue
			}

			// Incremental strategy: stop when we find an unchanged job
			if req.Strategy == "incremental" && found {
				if dbUpdatedTime == job.UpdatedTime {
					slog.Info("up to date point reached, stopping incremental scrape",
						"run_id", runID,
						"job_id", job.ID,
						"db_updated", dbUpdatedTime,
						"api_updated", job.UpdatedTime,
					)
					result.JobsSkipped++
					stopScraping = true
					result.StopReason = "incremental: up to date"
					break pageLoop
				}
			}

			// Determine if this is an insert or update
			isUpdate := found

			// Fetch full details for each job
			detail, err := r.client.FetchJobDetail(job.ID)
			if err != nil {
				errMsg := "job " + job.ID + ": " + err.Error()
				result.Errors = append(result.Errors, errMsg)
				slog.Error("failed to fetch job detail", "run_id", runID, "id", job.ID, "error", err)
				continue
			}

			// Store in database
			if err := r.store.UpsertJob(ctx, detail); err != nil {
				errMsg := "store " + job.ID + ": " + err.Error()
				result.Errors = append(result.Errors, errMsg)
				slog.Error("failed to store job", "run_id", runID, "id", job.ID, "error", err)
				continue
			}

			if isUpdate {
				result.JobsUpdated++
				slog.Debug("updated job", "run_id", runID, "id", job.ID)
			} else {
				result.JobsInserted++
				slog.Debug("inserted job", "run_id", runID, "id", job.ID)
			}

			// Send to AI service for processing (if configured)
			if r.aiClient != nil {
				r.processJobWithAI(ctx, job.ID, &result)
			}
		}
	}

	return result
}

// processJobWithAI sends a job to the AI service for processing.
func (r *Runner) processJobWithAI(ctx context.Context, jobID string, result *RunResult) {
	resp, err := r.aiClient.ProcessJob(ctx, jobID)
	if err != nil {
		slog.Warn("AI processing failed",
			"job_id", jobID,
			"error", err,
		)
		result.AIJobsFailed++
		return
	}

	if resp.Skipped {
		result.AIJobsSkipped++
		slog.Debug("AI processing skipped", "job_id", jobID, "reason", resp.SkipReason)
	} else {
		result.AIJobsProcessed++
		slog.Debug("AI processing completed", "job_id", jobID, "saved_to_db", resp.SavedToDB)
	}
}
