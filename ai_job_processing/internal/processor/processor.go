package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ai_job_processing/internal/gemini"
	"ai_job_processing/internal/models"
	"ai_job_processing/internal/store"
)

// Processor handles job processing operations.
type Processor struct {
	gemini          *gemini.Client
	store           *store.Store
	targetLanguages []string
}

// NewProcessor creates a new Processor.
func NewProcessor(geminiClient *gemini.Client, st *store.Store, targetLanguages []string) *Processor {
	return &Processor{
		gemini:          geminiClient,
		store:           st,
		targetLanguages: targetLanguages,
	}
}

// Process normalizes and translates raw job data (no database interaction).
func (p *Processor) Process(ctx context.Context, req *models.ProcessRequest) (*models.ProcessResponse, error) {
	start := time.Now()

	slog.Info("starting job processing (raw data)",
		"target_languages", p.targetLanguages,
	)

	// Normalize the job description
	normalized, err := p.gemini.NormalizeJobDescription(ctx, req.Title, req.Description, req.SourceLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	// Translate to all target languages
	translations, err := p.gemini.TranslateMultipleNormalized(ctx, req.Title, normalized, sourceLanguage, p.targetLanguages)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	slog.Info("job processing completed (raw data)",
		"translations", len(translations),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.ProcessResponse{
		Normalized:   normalized,
		Translations: translations,
		ProcessedAt:  time.Now(),
	}, nil
}

// ProcessByID loads a job from the database, processes it, and saves the results.
// Skips processing if job is already normalized and translated (unless Force is true).
func (p *Processor) ProcessByID(ctx context.Context, req *models.ProcessByIDRequest) (*models.ProcessResponse, error) {
	start := time.Now()

	slog.Info("starting job processing by ID",
		"job_id", req.JobID,
		"force", req.Force,
	)

	// Load job from database
	job, err := p.store.GetJobByID(ctx, req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to load job: %w", err)
	}

	if job.Title == "" || job.Description == "" {
		return nil, fmt.Errorf("job has no title or description: %s", req.JobID)
	}

	// Determine target languages
	targetLanguages := req.TargetLanguages
	if len(targetLanguages) == 0 {
		targetLanguages = p.targetLanguages
	}

	// Check existing translations (unless Force)
	if !req.Force {
		missing, skipped, err := p.store.GetMissingLanguages(ctx, req.JobID, targetLanguages)
		if err != nil {
			slog.Warn("failed to check existing languages", "error", err)
		} else if len(missing) == 0 && job.IsNormalized {
			// All requested languages already exist and job is normalized
			slog.Info("job already fully processed, skipping",
				"job_id", req.JobID,
				"skipped_languages", skipped,
			)

			normalized, translations, _ := p.store.GetExistingNormalizedContent(ctx, req.JobID)
			return &models.ProcessResponse{
				JobID:        req.JobID,
				Normalized:   normalized,
				Translations: translations,
				ProcessedAt:  time.Now(),
				Skipped:      true,
				SkipReason:   "already normalized and translated",
			}, nil
		} else if len(missing) < len(targetLanguages) {
			// Some languages exist, only process missing ones
			slog.Info("some translations exist, processing only missing",
				"job_id", req.JobID,
				"missing", missing,
				"skipped", skipped,
			)
			targetLanguages = missing
		}
	}

	// Determine source language
	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = job.Language
	}
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	// Normalize the job description
	normalized, err := p.gemini.NormalizeJobDescription(ctx, job.Title, job.Description, sourceLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	// Translate to all target languages
	translations, err := p.gemini.TranslateMultipleNormalized(ctx, job.Title, normalized, sourceLanguage, targetLanguages)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	// Save all translations to database
	savedToDB := false
	if err := p.store.SaveAllTranslations(ctx, req.JobID, translations, true); err != nil {
		slog.Error("failed to save translations",
			"job_id", req.JobID,
			"error", err,
		)
	} else {
		savedToDB = true
	}

	slog.Info("job processing by ID completed",
		"job_id", req.JobID,
		"translations", len(translations),
		"saved_to_db", savedToDB,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.ProcessResponse{
		JobID:        req.JobID,
		Normalized:   normalized,
		Translations: translations,
		ProcessedAt:  time.Now(),
		SavedToDB:    savedToDB,
	}, nil
}

// TranslateByID translates a job's raw description to target languages WITHOUT normalizing.
// The original job description is translated directly.
// Skips languages that already have translations (unless Force is true).
func (p *Processor) TranslateByID(ctx context.Context, req *models.TranslateByIDRequest) (*models.TranslateResponse, error) {
	start := time.Now()

	slog.Info("starting job translation by ID (no normalization)",
		"job_id", req.JobID,
		"force", req.Force,
	)

	// Load job from database
	job, err := p.store.GetJobByID(ctx, req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to load job: %w", err)
	}

	if job.Title == "" || job.Description == "" {
		return nil, fmt.Errorf("job has no title or description: %s", req.JobID)
	}

	// Determine target languages
	targetLanguages := req.TargetLanguages
	if len(targetLanguages) == 0 {
		targetLanguages = p.targetLanguages
	}

	var skippedLanguages []string

	// Check existing translations (unless Force)
	if !req.Force {
		missing, skipped, err := p.store.GetMissingLanguages(ctx, req.JobID, targetLanguages)
		if err != nil {
			slog.Warn("failed to check existing languages", "error", err)
		} else if len(missing) == 0 {
			// All requested languages already exist
			slog.Info("all translations already exist, skipping",
				"job_id", req.JobID,
				"skipped_languages", skipped,
			)

			translations, _ := p.store.GetExistingTranslations(ctx, req.JobID)
			return &models.TranslateResponse{
				JobID:            req.JobID,
				Translations:     translations,
				ProcessedAt:      time.Now(),
				Skipped:          true,
				SkipReason:       "all languages already translated",
				SkippedLanguages: skipped,
			}, nil
		} else {
			skippedLanguages = skipped
			targetLanguages = missing
			slog.Info("some translations exist, processing only missing",
				"job_id", req.JobID,
				"missing", missing,
				"skipped", skipped,
			)
		}
	}

	// Determine source language
	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = job.Language
	}
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	// Translate raw description to all target languages
	translations, err := p.gemini.TranslateMultipleRaw(ctx, job.Title, job.Description, sourceLanguage, targetLanguages)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	// Save translations to database (without normalization fields)
	savedToDB := false
	if err := p.store.SaveAllTranslations(ctx, req.JobID, translations, false); err != nil {
		slog.Error("failed to save translations",
			"job_id", req.JobID,
			"error", err,
		)
	} else {
		savedToDB = true
	}

	slog.Info("job translation by ID completed",
		"job_id", req.JobID,
		"translations", len(translations),
		"saved_to_db", savedToDB,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.TranslateResponse{
		JobID:            req.JobID,
		Translations:     translations,
		ProcessedAt:      time.Now(),
		SavedToDB:        savedToDB,
		SkippedLanguages: skippedLanguages,
	}, nil
}

// NormalizeByID normalizes a job from the database WITHOUT translating.
// Skips if already normalized (unless Force is true).
func (p *Processor) NormalizeByID(ctx context.Context, req *models.NormalizeByIDRequest) (*models.NormalizeResponse, error) {
	start := time.Now()

	slog.Info("starting job normalization by ID (no translation)",
		"job_id", req.JobID,
		"force", req.Force,
	)

	// Load job from database
	job, err := p.store.GetJobByID(ctx, req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to load job: %w", err)
	}

	if job.Title == "" || job.Description == "" {
		return nil, fmt.Errorf("job has no title or description: %s", req.JobID)
	}

	// Check if already normalized (unless Force)
	if !req.Force && job.IsNormalized {
		slog.Info("job already normalized, skipping",
			"job_id", req.JobID,
		)

		normalized, _, _ := p.store.GetExistingNormalizedContent(ctx, req.JobID)
		return &models.NormalizeResponse{
			JobID:          req.JobID,
			SourceLanguage: job.Language,
			Normalized:     normalized,
			ProcessedAt:    time.Now(),
			Skipped:        true,
			SkipReason:     "already normalized",
		}, nil
	}

	// Determine source language
	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = job.Language
	}
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	// Normalize the job description
	normalized, err := p.gemini.NormalizeJobDescription(ctx, job.Title, job.Description, sourceLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	// Save to database
	savedToDB := false
	if err := p.store.SaveNormalizedContent(ctx, req.JobID, job.Language, normalized); err != nil {
		slog.Error("failed to save normalized content",
			"job_id", req.JobID,
			"error", err,
		)
	} else {
		savedToDB = true
	}

	slog.Info("job normalization by ID completed",
		"job_id", req.JobID,
		"saved_to_db", savedToDB,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.NormalizeResponse{
		JobID:          req.JobID,
		SourceLanguage: sourceLanguage,
		Normalized:     normalized,
		ProcessedAt:    time.Now(),
		SavedToDB:      savedToDB,
	}, nil
}

// Normalize only normalizes raw job data (no translation, no database).
func (p *Processor) Normalize(ctx context.Context, req *models.NormalizeRequest) (*models.NormalizeResponse, error) {
	start := time.Now()

	slog.Info("starting job normalization (raw data)")

	normalized, err := p.gemini.NormalizeJobDescription(ctx, req.Title, req.Description, req.SourceLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	slog.Info("job normalization completed",
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.NormalizeResponse{
		SourceLanguage: sourceLanguage,
		Normalized:     normalized,
		ProcessedAt:    time.Now(),
	}, nil
}

// Translate translates raw job data (supports both normalized and raw description).
func (p *Processor) Translate(ctx context.Context, req *models.TranslateRequest) (*models.TranslateResponse, error) {
	start := time.Now()

	targetLangs := req.TargetLanguages
	if len(targetLangs) == 0 {
		targetLangs = p.targetLanguages
	}

	slog.Info("starting translation (raw data)",
		"target_languages", targetLangs,
	)

	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = "de"
	}

	var translations []models.TranslatedContent
	var err error

	// Check if we have normalized content or just raw description
	if req.Normalized != nil && (req.Normalized.Tasks != "" || req.Normalized.Requirements != "" || req.Normalized.Offer != "") {
		// Translate normalized content
		translations, err = p.gemini.TranslateMultipleNormalized(ctx, req.Title, req.Normalized, sourceLanguage, targetLangs)
	} else if req.Description != "" {
		// Translate raw description
		translations, err = p.gemini.TranslateMultipleRaw(ctx, req.Title, req.Description, sourceLanguage, targetLangs)
	} else {
		return nil, fmt.Errorf("either normalized content or description is required")
	}

	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	slog.Info("translation completed",
		"translations", len(translations),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.TranslateResponse{
		Translations: translations,
		ProcessedAt:  time.Now(),
	}, nil
}

// GetTargetLanguages returns the configured target languages.
func (p *Processor) GetTargetLanguages() []string {
	return p.targetLanguages
}

// GetPendingJobsCount returns the count of jobs pending normalization.
func (p *Processor) GetPendingJobsCount(ctx context.Context) (int, error) {
	if p.store == nil {
		return 0, nil
	}
	return p.store.CountPendingJobs(ctx)
}

// GetPendingJobs returns jobs pending normalization.
func (p *Processor) GetPendingJobs(ctx context.Context, limit int) ([]models.JobFromDB, error) {
	if p.store == nil {
		return nil, nil
	}
	return p.store.GetPendingJobs(ctx, limit)
}
