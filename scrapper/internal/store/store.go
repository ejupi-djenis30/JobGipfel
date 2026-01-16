package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"scrapper/internal/models"
)

// Store handles database operations for job data.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new Store instance.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// DB returns the underlying database connection.
func (s *Store) DB() *sqlx.DB {
	return s.db
}

// UpsertJob inserts or updates a job and all its related child records.
// Uses a transaction to ensure ACID compliance.
// Companies and locations are deduplicated by unique key lookup.
func (s *Store) UpsertJob(ctx context.Context, job *models.JobDetail) error {
	// Step 1: Marshal job to JSON for raw_data storage
	rawData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job to JSON: %w", err)
	}

	// Step 2: Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Step 3: Get or create company (deduplicated)
	companyID, err := getOrCreateCompany(ctx, tx, &job.JobContent.Company)
	if err != nil {
		return fmt.Errorf("failed to get/create company: %w", err)
	}

	// Step 4: Get or create location (deduplicated)
	locationID, err := getOrCreateLocation(ctx, tx, &job.JobContent.Location)
	if err != nil {
		return fmt.Errorf("failed to get/create location: %w", err)
	}

	// Step 5: Upsert parent job record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO jobs (
			id, source, created_time, updated_time, status, source_system,
			external_ref, stellennummer_egov, fingerprint, reporting_obligation,
			raw_data, company_id, location_id, created_at, updated_at
		) VALUES ($1, 'jobroom', $2, $3, 'active', $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			updated_time = EXCLUDED.updated_time,
			status = EXCLUDED.status,
			source_system = EXCLUDED.source_system,
			external_ref = EXCLUDED.external_ref,
			stellennummer_egov = EXCLUDED.stellennummer_egov,
			fingerprint = EXCLUDED.fingerprint,
			reporting_obligation = EXCLUDED.reporting_obligation,
			raw_data = EXCLUDED.raw_data,
			company_id = EXCLUDED.company_id,
			location_id = EXCLUDED.location_id,
			updated_at = NOW()`,
		job.ID,
		job.CreatedTime,
		job.UpdatedTime,
		job.SourceSystem,
		job.ExternalReference,
		job.StellennummerEgov,
		job.Fingerprint,
		job.ReportingObligation,
		string(rawData),
		companyID,
		locationID,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert job: %w", err)
	}

	// Step 6: Clean child tables
	childTables := []string{
		"employments",
		"publications",
		"apply_channels",
		"job_descriptions",
		"occupations",
	}
	for _, table := range childTables {
		_, err = tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE job_id = $1", table), job.ID)
		if err != nil {
			return fmt.Errorf("failed to clean %s: %w", table, err)
		}
	}

	// Step 7: Insert child records

	// 7a: Insert employment
	_, err = tx.ExecContext(ctx, `
		INSERT INTO employments (
			job_id, start_date, end_date, short_employment,
			immediately, permanent, workload_min, workload_max, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		job.ID,
		parseNullableDate(job.JobContent.Employment.StartDate),
		parseNullableDate(job.JobContent.Employment.EndDate),
		job.JobContent.Employment.ShortEmployment,
		job.JobContent.Employment.Immediately,
		job.JobContent.Employment.Permanent,
		parseWorkload(job.JobContent.Employment.WorkloadPercentageMin),
		parseWorkload(job.JobContent.Employment.WorkloadPercentageMax),
	)
	if err != nil {
		return fmt.Errorf("failed to insert employment: %w", err)
	}

	// 7b: Insert publication
	_, err = tx.ExecContext(ctx, `
		INSERT INTO publications (
			job_id, start_date, end_date, eures_display,
			public_display, restricted_display, company_anonymous, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		job.ID,
		parseDate(job.Publication.StartDate),
		parseDate(job.Publication.EndDate),
		job.Publication.EuresDisplay,
		job.Publication.PublicDisplay,
		job.Publication.RestrictedDisplay,
		job.Publication.CompanyAnonymous,
	)
	if err != nil {
		return fmt.Errorf("failed to insert publication: %w", err)
	}

	// 7c: Insert apply channel
	_, err = tx.ExecContext(ctx, `
		INSERT INTO apply_channels (
			job_id, raw_post_address, post_address, email_address,
			phone_number, form_url, additional_info, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		job.ID,
		job.JobContent.ApplyChannel.RawPostAddress,
		job.JobContent.ApplyChannel.PostAddress,
		job.JobContent.ApplyChannel.EmailAddress,
		job.JobContent.ApplyChannel.PhoneNumber,
		job.JobContent.ApplyChannel.FormURL,
		job.JobContent.ApplyChannel.AdditionalInfo,
	)
	if err != nil {
		return fmt.Errorf("failed to insert apply_channel: %w", err)
	}

	// 7d: Insert job descriptions (one-to-many)
	for _, desc := range job.JobContent.JobDescriptions {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO job_descriptions (
				job_id, language_iso_code, title, description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (job_id, language_iso_code) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				updated_at = NOW()`,
			job.ID,
			desc.LanguageIsoCode,
			desc.Title,
			desc.Description,
		)
		if err != nil {
			return fmt.Errorf("failed to insert job_description: %w", err)
		}
	}

	// 7e: Insert occupations (one-to-many)
	for _, occ := range job.JobContent.Occupations {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO occupations (
				job_id, avam_occupation_code, work_experience,
				education_code, qualification_code, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
			job.ID,
			occ.AvamOccupationCode,
			occ.WorkExperience,
			occ.EducationCode,
			occ.QualificationCode,
		)
		if err != nil {
			return fmt.Errorf("failed to insert occupation: %w", err)
		}
	}

	// Step 8: Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getOrCreateCompany finds an existing company by unique key or creates a new one.
func getOrCreateCompany(ctx context.Context, tx *sqlx.Tx, company *models.Company) (int64, error) {
	// Try to find existing company by unique key (name + postal_code + city)
	var id int64
	err := tx.GetContext(ctx, &id, `
		SELECT id FROM companies 
		WHERE name = $1 AND COALESCE(postal_code, '') = $2 AND COALESCE(city, '') = $3`,
		company.Name, nullableString(company.PostalCode), nullableString(company.City),
	)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to lookup company: %w", err)
	}

	// Insert new company
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO companies (
			name, street, house_number, postal_code, city,
			country_iso_code, phone, email, website, surrogate, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id`,
		company.Name,
		company.Street,
		company.HouseNumber,
		company.PostalCode,
		company.City,
		company.CountryIsoCode,
		company.Phone,
		company.Email,
		company.Website,
		company.Surrogate,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert company: %w", err)
	}

	return id, nil
}

// getOrCreateLocation finds an existing location by unique key or creates a new one.
func getOrCreateLocation(ctx context.Context, tx *sqlx.Tx, location *models.Location) (int64, error) {
	// Try to find existing location by unique key (postal_code + city + canton_code)
	var id int64
	err := tx.GetContext(ctx, &id, `
		SELECT id FROM locations 
		WHERE COALESCE(postal_code, '') = $1 AND COALESCE(city, '') = $2 AND COALESCE(canton_code, '') = $3`,
		nullableString(location.PostalCode), nullableString(location.City), nullableString(location.CantonCode),
	)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to lookup location: %w", err)
	}

	// Insert new location
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO locations (
			remarks, city, postal_code, communal_code,
			region_code, canton_code, country_iso_code, lat, lon, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id`,
		location.Remarks,
		location.City,
		location.PostalCode,
		location.CommunalCode,
		location.RegionCode,
		location.CantonCode,
		location.CountryIsoCode,
		parseCoordinate(location.Coordinates.Lat),
		parseCoordinate(location.Coordinates.Lon),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert location: %w", err)
	}

	return id, nil
}

// GetJobLastUpdated retrieves the updated_time for a job by ID.
func (s *Store) GetJobLastUpdated(ctx context.Context, id string) (string, bool, error) {
	var updatedTime string
	err := s.db.GetContext(ctx, &updatedTime, "SELECT updated_time FROM jobs WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get job updated_time: %w", err)
	}
	return updatedTime, true, nil
}

// ListJobs returns a paginated list of job summaries.
func (s *Store) ListJobs(ctx context.Context, limit, offset int) ([]models.JobSummary, error) {
	var jobs []models.JobSummary
	err := s.db.SelectContext(ctx, &jobs,
		`SELECT id, status, source, created_time, updated_time 
		 FROM jobs ORDER BY updated_time DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	return jobs, nil
}

// CountJobs returns the total count of jobs in the database.
func (s *Store) CountJobs(ctx context.Context) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM jobs")
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}
	return count, nil
}

// CreateRun creates a new scrape run and returns its ID.
func (s *Store) CreateRun(ctx context.Context, strategy string, filters string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO scrape_runs (strategy, start_time, status, filters, created_at, updated_at)
		VALUES ($1, NOW(), 'running', $2, NOW(), NOW())
		RETURNING id`,
		strategy, nullableStringPtr(filters),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create run: %w", err)
	}
	return id, nil
}

// UpdateRun updates a scrape run with final status and metrics.
func (s *Store) UpdateRun(ctx context.Context, runID int64, status string, processed, inserted, updated, skipped, pagesScraped int, errLog string) error {
	var errLogPtr *string
	if errLog != "" {
		errLogPtr = &errLog
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE scrape_runs
		SET status = $1, end_time = NOW(),
		    jobs_processed = $2, jobs_inserted = $3, jobs_updated = $4, jobs_skipped = $5,
		    pages_scraped = $6, error_log = $7, updated_at = NOW()
		WHERE id = $8`,
		status, processed, inserted, updated, skipped, pagesScraped, errLogPtr, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	return nil
}

// ListRuns returns the most recent scrape runs.
func (s *Store) ListRuns(ctx context.Context, limit int) ([]models.ScrapeRun, error) {
	var runs []models.ScrapeRun
	err := s.db.SelectContext(ctx, &runs,
		`SELECT id, strategy, start_time, end_time, status, jobs_processed, 
		        jobs_inserted, jobs_updated, jobs_skipped, pages_scraped, filters, error_log 
		 FROM scrape_runs ORDER BY start_time DESC LIMIT $1`,
		limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	return runs, nil
}

// GetRun retrieves a specific run by ID.
func (s *Store) GetRun(ctx context.Context, id int64) (*models.ScrapeRun, error) {
	var run models.ScrapeRun
	err := s.db.GetContext(ctx, &run,
		`SELECT id, strategy, start_time, end_time, status, jobs_processed, 
		        jobs_inserted, jobs_updated, jobs_skipped, pages_scraped, filters, error_log 
		 FROM scrape_runs WHERE id = $1`,
		id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return &run, nil
}

// Helper functions

func nullableString(s string) string {
	return s
}

func nullableStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	// Try multiple date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

func parseNullableDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return parseDate(*s)
}

func parseWorkload(s string) *int {
	if s == "" {
		return nil
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
		return &v
	}
	return nil
}

func parseCoordinate(s string) *float64 {
	if s == "" {
		return nil
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err == nil {
		return &v
	}
	return nil
}
