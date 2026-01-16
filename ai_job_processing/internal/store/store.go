package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"ai_job_processing/internal/models"
)

// Store handles database operations.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new Store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// DB returns the underlying database connection.
func (s *Store) DB() *sqlx.DB {
	return s.db
}

// GetJobByID retrieves a job and its primary description by job ID.
func (s *Store) GetJobByID(ctx context.Context, jobID string) (*models.JobFromDB, error) {
	var job models.JobFromDB

	err := s.db.GetContext(ctx, &job, `
		SELECT 
			j.id,
			COALESCE(jd.title, '') as title,
			COALESCE(jd.description, '') as description,
			COALESCE(jd.language_iso_code, 'de') as language_iso_code,
			COALESCE(jd.is_normalized, FALSE) as is_normalized
		FROM jobs j
		LEFT JOIN job_descriptions jd ON j.id = jd.job_id
		WHERE j.id = $1
		ORDER BY jd.id ASC
		LIMIT 1`,
		jobID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// IsJobNormalized checks if a job has already been normalized.
func (s *Store) IsJobNormalized(ctx context.Context, jobID string) (bool, error) {
	var isNormalized bool
	err := s.db.GetContext(ctx, &isNormalized, `
		SELECT COALESCE(is_normalized, FALSE)
		FROM job_descriptions
		WHERE job_id = $1
		ORDER BY id ASC
		LIMIT 1`,
		jobID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check normalization status: %w", err)
	}
	return isNormalized, nil
}

// GetExistingLanguages returns a list of languages that already have translations for a job.
func (s *Store) GetExistingLanguages(ctx context.Context, jobID string) ([]string, error) {
	var languages []string
	err := s.db.SelectContext(ctx, &languages, `
		SELECT language_iso_code
		FROM job_descriptions
		WHERE job_id = $1`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing languages: %w", err)
	}
	return languages, nil
}

// GetMissingLanguages returns which of the requested languages don't have translations yet.
func (s *Store) GetMissingLanguages(ctx context.Context, jobID string, targetLanguages []string) ([]string, []string, error) {
	existing, err := s.GetExistingLanguages(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	existingMap := make(map[string]bool)
	for _, lang := range existing {
		existingMap[lang] = true
	}

	var missing, skipped []string
	for _, lang := range targetLanguages {
		if existingMap[lang] {
			skipped = append(skipped, lang)
		} else {
			missing = append(missing, lang)
		}
	}

	return missing, skipped, nil
}

// GetExistingNormalizedContent retrieves already-normalized content for a job.
func (s *Store) GetExistingNormalizedContent(ctx context.Context, jobID string) (*models.NormalizedContent, []models.TranslatedContent, error) {
	var normalized struct {
		Tasks        sql.NullString `db:"tasks"`
		Requirements sql.NullString `db:"requirements"`
		Offer        sql.NullString `db:"offer"`
	}

	err := s.db.GetContext(ctx, &normalized, `
		SELECT tasks, requirements, offer
		FROM job_descriptions
		WHERE job_id = $1 AND is_normalized = TRUE
		ORDER BY id ASC
		LIMIT 1`,
		jobID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get normalized content: %w", err)
	}

	normalizedContent := &models.NormalizedContent{
		Tasks:        normalized.Tasks.String,
		Requirements: normalized.Requirements.String,
		Offer:        normalized.Offer.String,
	}

	// Get all translations
	var translations []struct {
		Language     string         `db:"language_iso_code"`
		Title        string         `db:"title"`
		Description  string         `db:"description"`
		Tasks        sql.NullString `db:"tasks"`
		Requirements sql.NullString `db:"requirements"`
		Offer        sql.NullString `db:"offer"`
	}

	err = s.db.SelectContext(ctx, &translations, `
		SELECT language_iso_code, title, description, tasks, requirements, offer
		FROM job_descriptions
		WHERE job_id = $1
		ORDER BY language_iso_code`,
		jobID,
	)
	if err != nil {
		return normalizedContent, nil, nil
	}

	translatedContent := make([]models.TranslatedContent, len(translations))
	for i, t := range translations {
		translatedContent[i] = models.TranslatedContent{
			Language:     t.Language,
			Title:        t.Title,
			Description:  t.Description,
			Tasks:        t.Tasks.String,
			Requirements: t.Requirements.String,
			Offer:        t.Offer.String,
		}
	}

	return normalizedContent, translatedContent, nil
}

// GetExistingTranslations retrieves all existing translations for a job.
func (s *Store) GetExistingTranslations(ctx context.Context, jobID string) ([]models.TranslatedContent, error) {
	var translations []struct {
		Language     string         `db:"language_iso_code"`
		Title        string         `db:"title"`
		Description  string         `db:"description"`
		Tasks        sql.NullString `db:"tasks"`
		Requirements sql.NullString `db:"requirements"`
		Offer        sql.NullString `db:"offer"`
	}

	err := s.db.SelectContext(ctx, &translations, `
		SELECT language_iso_code, title, description, tasks, requirements, offer
		FROM job_descriptions
		WHERE job_id = $1
		ORDER BY language_iso_code`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get translations: %w", err)
	}

	result := make([]models.TranslatedContent, len(translations))
	for i, t := range translations {
		result[i] = models.TranslatedContent{
			Language:     t.Language,
			Title:        t.Title,
			Description:  t.Description,
			Tasks:        t.Tasks.String,
			Requirements: t.Requirements.String,
			Offer:        t.Offer.String,
		}
	}

	return result, nil
}

// GetJobDescriptions retrieves all descriptions for a job.
func (s *Store) GetJobDescriptions(ctx context.Context, jobID string) ([]models.JobFromDB, error) {
	var descriptions []models.JobFromDB

	err := s.db.SelectContext(ctx, &descriptions, `
		SELECT 
			job_id as id,
			COALESCE(title, '') as title,
			COALESCE(description, '') as description,
			COALESCE(language_iso_code, 'de') as language_iso_code,
			COALESCE(is_normalized, FALSE) as is_normalized
		FROM job_descriptions
		WHERE job_id = $1
		ORDER BY language_iso_code`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get job descriptions: %w", err)
	}

	return descriptions, nil
}

// SaveNormalizedContent saves only the normalized content (tasks, requirements, offer) for a job.
func (s *Store) SaveNormalizedContent(ctx context.Context, jobID, language string, normalized *models.NormalizedContent) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE job_descriptions
		SET 
			tasks = $1,
			requirements = $2,
			offer = $3,
			is_normalized = TRUE,
			normalized_at = $4,
			updated_at = NOW()
		WHERE job_id = $5 AND language_iso_code = $6`,
		normalized.Tasks,
		normalized.Requirements,
		normalized.Offer,
		time.Now(),
		jobID,
		language,
	)
	if err != nil {
		return fmt.Errorf("failed to save normalized content: %w", err)
	}

	return nil
}

// SaveTranslation saves a single translation (without normalization fields).
func (s *Store) SaveTranslation(ctx context.Context, jobID string, translated *models.TranslatedContent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO job_descriptions (
			job_id, language_iso_code, title, description,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (job_id, language_iso_code) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			updated_at = NOW()`,
		jobID,
		translated.Language,
		translated.Title,
		translated.Description,
	)
	if err != nil {
		return fmt.Errorf("failed to save translation: %w", err)
	}

	return nil
}

// SaveNormalizedTranslation saves a translation with normalization fields.
func (s *Store) SaveNormalizedTranslation(ctx context.Context, jobID string, translated *models.TranslatedContent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO job_descriptions (
			job_id, language_iso_code, title, description,
			tasks, requirements, offer, is_normalized, normalized_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, NOW(), NOW())
		ON CONFLICT (job_id, language_iso_code) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			tasks = EXCLUDED.tasks,
			requirements = EXCLUDED.requirements,
			offer = EXCLUDED.offer,
			is_normalized = TRUE,
			normalized_at = EXCLUDED.normalized_at,
			updated_at = NOW()`,
		jobID,
		translated.Language,
		translated.Title,
		translated.Description,
		translated.Tasks,
		translated.Requirements,
		translated.Offer,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to save normalized translation: %w", err)
	}

	return nil
}

// SaveAllTranslations saves all translations for a job (with normalization).
func (s *Store) SaveAllTranslations(ctx context.Context, jobID string, translations []models.TranslatedContent, normalized bool) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, t := range translations {
		if normalized {
			// Save with normalization fields
			_, err := tx.ExecContext(ctx, `
				INSERT INTO job_descriptions (
					job_id, language_iso_code, title, description,
					tasks, requirements, offer, is_normalized, normalized_at,
					created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, NOW(), NOW())
				ON CONFLICT (job_id, language_iso_code) DO UPDATE SET
					title = EXCLUDED.title,
					description = EXCLUDED.description,
					tasks = EXCLUDED.tasks,
					requirements = EXCLUDED.requirements,
					offer = EXCLUDED.offer,
					is_normalized = TRUE,
					normalized_at = EXCLUDED.normalized_at,
					updated_at = NOW()`,
				jobID,
				t.Language,
				t.Title,
				t.Description,
				t.Tasks,
				t.Requirements,
				t.Offer,
				time.Now(),
			)
			if err != nil {
				return fmt.Errorf("failed to save translation for %s: %w", t.Language, err)
			}
		} else {
			// Save without normalization fields (simple translation)
			_, err := tx.ExecContext(ctx, `
				INSERT INTO job_descriptions (
					job_id, language_iso_code, title, description,
					created_at, updated_at
				) VALUES ($1, $2, $3, $4, NOW(), NOW())
				ON CONFLICT (job_id, language_iso_code) DO UPDATE SET
					title = EXCLUDED.title,
					description = EXCLUDED.description,
					updated_at = NOW()`,
				jobID,
				t.Language,
				t.Title,
				t.Description,
			)
			if err != nil {
				return fmt.Errorf("failed to save translation for %s: %w", t.Language, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetPendingJobs retrieves jobs that haven't been normalized yet.
func (s *Store) GetPendingJobs(ctx context.Context, limit int) ([]models.JobFromDB, error) {
	var jobs []models.JobFromDB

	err := s.db.SelectContext(ctx, &jobs, `
		SELECT DISTINCT
			j.id,
			COALESCE(jd.title, '') as title,
			COALESCE(jd.description, '') as description,
			COALESCE(jd.language_iso_code, 'de') as language_iso_code,
			COALESCE(jd.is_normalized, FALSE) as is_normalized
		FROM jobs j
		INNER JOIN job_descriptions jd ON j.id = jd.job_id
		WHERE jd.is_normalized = FALSE OR jd.is_normalized IS NULL
		ORDER BY j.created_time DESC
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	return jobs, nil
}

// CountPendingJobs counts jobs that haven't been normalized.
func (s *Store) CountPendingJobs(ctx context.Context) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(DISTINCT job_id)
		FROM job_descriptions
		WHERE is_normalized = FALSE OR is_normalized IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending jobs: %w", err)
	}
	return count, nil
}
