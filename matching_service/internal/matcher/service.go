package matcher

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"matching_service/internal/auth"
	"matching_service/internal/gemini"
	"matching_service/internal/models"
)

// Service handles job matching.
type Service struct {
	db           *sqlx.DB
	authClient   *auth.Client
	geminiClient *gemini.Client
}

// NewService creates a new matching service.
func NewService(db *sqlx.DB, authClient *auth.Client, geminiClient *gemini.Client) *Service {
	return &Service{
		db:           db,
		authClient:   authClient,
		geminiClient: geminiClient,
	}
}

// GetMatches finds jobs matching the user's profile.
func (s *Service) GetMatches(ctx context.Context, accessToken string, req *models.MatchRequest) (*models.MatchResponse, error) {
	start := time.Now()

	// Get user profile
	profile, err := s.authClient.GetUserProfile(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	limit := 20
	if req != nil && req.Limit > 0 && req.Limit <= 100 {
		limit = req.Limit
	}

	minScore := 30.0 // Default minimum score
	if req != nil && req.MinScore > 0 {
		minScore = req.MinScore
	}

	// Get active jobs
	jobs, err := s.fetchJobs(ctx, limit*3) // Fetch more to filter by score
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jobs: %w", err)
	}

	// Score each job
	var matchedJobs []models.MatchedJob
	for _, job := range jobs {
		score := s.scoreJob(ctx, profile, &job, req)
		if score.MatchScore >= minScore {
			job.MatchScore = score.MatchScore
			job.SkillsMatch = score.SkillsMatch
			job.ExperienceMatch = score.ExperienceMatch
			job.LocationMatch = score.LocationMatch
			job.WorkloadMatch = score.WorkloadMatch
			job.MatchedSkills = score.MatchedSkills
			job.MissingSkills = score.MissingSkills
			job.MatchReasons = score.MatchReasons
			matchedJobs = append(matchedJobs, job)
		}

		if len(matchedJobs) >= limit {
			break
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(matchedJobs)-1; i++ {
		for j := i + 1; j < len(matchedJobs); j++ {
			if matchedJobs[j].MatchScore > matchedJobs[i].MatchScore {
				matchedJobs[i], matchedJobs[j] = matchedJobs[j], matchedJobs[i]
			}
		}
	}

	slog.Info("Matching completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"jobs_matched", len(matchedJobs),
	)

	return &models.MatchResponse{
		Jobs:      matchedJobs,
		Total:     len(matchedJobs),
		ProfileID: profile.UserID,
		MatchedAt: time.Now(),
	}, nil
}

// ScoreJob scores a specific job for the user.
func (s *Service) ScoreJob(ctx context.Context, accessToken, jobID string) (*models.JobScoreResponse, error) {
	// Get user profile
	profile, err := s.authClient.GetUserProfile(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Get job details
	job, err := s.fetchJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("job not found")
	}

	// Use AI for detailed scoring
	if s.geminiClient != nil {
		score, err := s.geminiClient.ScoreJobMatch(ctx, profile, job.Title, job.Description)
		if err == nil {
			score.JobID = jobID
			score.JobTitle = job.Title
			return score, nil
		}
		slog.Warn("AI scoring failed, using rule-based", "error", err)
	}

	// Fallback to rule-based scoring
	scoreResult := s.scoreJob(ctx, profile, job, nil)
	return &models.JobScoreResponse{
		JobID:           jobID,
		JobTitle:        job.Title,
		MatchScore:      scoreResult.MatchScore,
		SkillsMatch:     scoreResult.SkillsMatch,
		ExperienceMatch: scoreResult.ExperienceMatch,
		LocationMatch:   scoreResult.LocationMatch,
		WorkloadMatch:   scoreResult.WorkloadMatch,
		MatchedSkills:   scoreResult.MatchedSkills,
		MissingSkills:   scoreResult.MissingSkills,
	}, nil
}

// AnalyzeProfile analyzes the user's profile.
func (s *Service) AnalyzeProfile(ctx context.Context, accessToken string) (*models.ProfileAnalysis, error) {
	profile, err := s.authClient.GetUserProfile(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if s.geminiClient != nil {
		return s.geminiClient.AnalyzeProfile(ctx, profile)
	}

	// Basic analysis without AI
	skillNames := make([]string, len(profile.Skills))
	for i, s := range profile.Skills {
		skillNames[i] = s.Name
	}

	return &models.ProfileAnalysis{
		SkillsExtracted: skillNames,
		ExperienceYears: len(profile.Experiences),
		SeniorityLevel:  "mid",
		ProfileStrength: calculateProfileStrength(profile),
	}, nil
}

// scoreJob calculates match score using rule-based algorithm.
func (s *Service) scoreJob(ctx context.Context, profile *models.UserProfile, job *models.MatchedJob, req *models.MatchRequest) *matchScore {
	score := &matchScore{}

	// Skills matching (40% weight)
	userSkills := make(map[string]bool)
	for _, skill := range profile.Skills {
		userSkills[strings.ToLower(skill.Name)] = true
	}

	jobText := strings.ToLower(job.Title + " " + job.Description)
	for skill := range userSkills {
		if strings.Contains(jobText, skill) {
			score.MatchedSkills = append(score.MatchedSkills, skill)
		}
	}

	if len(profile.Skills) > 0 {
		score.SkillsMatch = float64(len(score.MatchedSkills)) / float64(len(profile.Skills)) * 100
	}

	// Experience matching (25% weight)
	score.ExperienceMatch = 70.0 // Default moderate match
	if len(profile.Experiences) > 3 {
		score.ExperienceMatch = 85.0
	} else if len(profile.Experiences) == 0 {
		score.ExperienceMatch = 40.0
	}

	// Location matching (15% weight)
	score.LocationMatch = 50.0 // Default
	if job.Location != nil && profile.City != "" {
		if strings.EqualFold(job.Location.City, profile.City) {
			score.LocationMatch = 100.0
			score.MatchReasons = append(score.MatchReasons, "Same city as your profile")
		} else if job.Location.Country == profile.Country {
			score.LocationMatch = 70.0
		}
	}

	// Workload matching (10% weight)
	score.WorkloadMatch = 80.0 // Default good match

	// Semantic match placeholder (10% weight)
	semanticMatch := 60.0

	// Calculate overall score
	score.MatchScore = score.SkillsMatch*0.40 +
		score.ExperienceMatch*0.25 +
		score.LocationMatch*0.15 +
		score.WorkloadMatch*0.10 +
		semanticMatch*0.10

	// Add match reasons
	if score.SkillsMatch >= 50 {
		score.MatchReasons = append(score.MatchReasons, fmt.Sprintf("%d of your skills match", len(score.MatchedSkills)))
	}

	return score
}

type matchScore struct {
	MatchScore      float64
	SkillsMatch     float64
	ExperienceMatch float64
	LocationMatch   float64
	WorkloadMatch   float64
	MatchedSkills   []string
	MissingSkills   []string
	MatchReasons    []string
}

func (s *Service) fetchJobs(ctx context.Context, limit int) ([]models.MatchedJob, error) {
	query := `
		SELECT
			j.id,
			jd.title,
			LEFT(jd.description, 500) as description,
			c.id as company_id,
			c.name as company_name,
			c.city as company_city,
			l.city as location_city,
			l.canton_code as location_canton,
			l.country_iso_code as location_country,
			CAST(l.lat AS FLOAT) as location_lat,
			CAST(l.lon AS FLOAT) as location_lon,
			e.permanent,
			e.immediately,
			e.workload_min,
			e.workload_max,
			j.external_url,
			j.source,
			j.created_time
		FROM jobs j
		LEFT JOIN job_descriptions jd ON j.id = jd.job_id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN locations l ON j.location_id = l.id
		LEFT JOIN employment e ON j.id = e.job_id
		WHERE j.status = 'active'
		ORDER BY j.created_time DESC
		LIMIT $1
	`

	rows, err := s.db.QueryxContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.MatchedJob
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (s *Service) fetchJobByID(ctx context.Context, id string) (*models.MatchedJob, error) {
	query := `
		SELECT
			j.id,
			jd.title,
			jd.description,
			c.id as company_id,
			c.name as company_name,
			c.city as company_city,
			l.city as location_city,
			l.canton_code as location_canton,
			l.country_iso_code as location_country,
			CAST(l.lat AS FLOAT) as location_lat,
			CAST(l.lon AS FLOAT) as location_lon,
			e.permanent,
			e.immediately,
			e.workload_min,
			e.workload_max,
			j.external_url,
			j.source,
			j.created_time
		FROM jobs j
		LEFT JOIN job_descriptions jd ON j.id = jd.job_id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN locations l ON j.location_id = l.id
		LEFT JOIN employment e ON j.id = e.job_id
		WHERE j.id = $1
		LIMIT 1
	`

	rows, err := s.db.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanJob(rows)
	}
	return nil, nil
}

func scanJob(rows *sqlx.Rows) (*models.MatchedJob, error) {
	var job struct {
		ID              string          `db:"id"`
		Title           sql.NullString  `db:"title"`
		Description     sql.NullString  `db:"description"`
		CompanyID       sql.NullInt64   `db:"company_id"`
		CompanyName     sql.NullString  `db:"company_name"`
		CompanyCity     sql.NullString  `db:"company_city"`
		LocationCity    sql.NullString  `db:"location_city"`
		LocationCanton  sql.NullString  `db:"location_canton"`
		LocationCountry sql.NullString  `db:"location_country"`
		LocationLat     sql.NullFloat64 `db:"location_lat"`
		LocationLon     sql.NullFloat64 `db:"location_lon"`
		Permanent       sql.NullBool    `db:"permanent"`
		Immediately     sql.NullBool    `db:"immediately"`
		WorkloadMin     sql.NullString  `db:"workload_min"`
		WorkloadMax     sql.NullString  `db:"workload_max"`
		ExternalURL     sql.NullString  `db:"external_url"`
		Source          string          `db:"source"`
		CreatedTime     time.Time       `db:"created_time"`
	}

	if err := rows.StructScan(&job); err != nil {
		return nil, err
	}

	result := &models.MatchedJob{
		ID:          job.ID,
		Title:       job.Title.String,
		Description: job.Description.String,
		Source:      job.Source,
		CreatedTime: job.CreatedTime,
		ExternalURL: job.ExternalURL.String,
	}

	if job.CompanyName.Valid {
		result.Company = &models.CompanyInfo{
			ID:   job.CompanyID.Int64,
			Name: job.CompanyName.String,
			City: job.CompanyCity.String,
		}
	}

	if job.LocationCity.Valid {
		result.Location = &models.LocationInfo{
			City:    job.LocationCity.String,
			Canton:  job.LocationCanton.String,
			Country: job.LocationCountry.String,
			Lat:     job.LocationLat.Float64,
			Lon:     job.LocationLon.Float64,
		}
	}

	result.Employment = &models.EmploymentInfo{
		Permanent:   job.Permanent.Bool,
		Immediately: job.Immediately.Bool,
		WorkloadMin: job.WorkloadMin.String,
		WorkloadMax: job.WorkloadMax.String,
	}

	return result, nil
}

func calculateProfileStrength(profile *models.UserProfile) int {
	score := 0
	if profile.FirstName != "" {
		score += 10
	}
	if profile.Headline != "" {
		score += 15
	}
	if profile.Summary != "" {
		score += 20
	}
	if len(profile.Skills) > 0 {
		score += min(len(profile.Skills)*5, 25)
	}
	if len(profile.Experiences) > 0 {
		score += min(len(profile.Experiences)*10, 30)
	}
	return min(score, 100)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
