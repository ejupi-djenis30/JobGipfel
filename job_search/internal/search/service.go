package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"job_search/internal/gemini"
	"job_search/internal/models"
)

// Service handles job search operations.
type Service struct {
	db           *sqlx.DB
	geminiClient *gemini.Client
}

// NewService creates a new search service.
func NewService(db *sqlx.DB, geminiClient *gemini.Client) *Service {
	return &Service{
		db:           db,
		geminiClient: geminiClient,
	}
}

// Search performs a job search with filters.
func (s *Service) Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	start := time.Now()

	// Default pagination
	page := 1
	perPage := 20
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 && req.Pagination.PerPage <= 100 {
			perPage = req.Pagination.PerPage
		}
	}
	offset := (page - 1) * perPage

	// Build query
	query, args, aiEnhanced := s.buildSearchQuery(ctx, req, perPage, offset)

	// Execute search
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var jobs []models.JobResult
	for rows.Next() {
		job, err := s.scanJobResult(rows)
		if err != nil {
			slog.Warn("Failed to scan job", "error", err)
			continue
		}
		jobs = append(jobs, *job)
	}

	// Get total count
	total, err := s.countResults(ctx, req)
	if err != nil {
		slog.Warn("Failed to count results", "error", err)
		total = len(jobs)
	}

	totalPages := (total + perPage - 1) / perPage

	slog.Info("Search completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"query", req.Query,
		"results", len(jobs),
		"total", total,
	)

	return &models.SearchResponse{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		Query:      req.Query,
		AIEnhanced: aiEnhanced,
	}, nil
}

// buildSearchQuery constructs the SQL query with filters.
func (s *Service) buildSearchQuery(ctx context.Context, req *models.SearchRequest, limit, offset int) (string, []interface{}, bool) {
	var conditions []string
	var args []interface{}
	argNum := 1
	aiEnhanced := false

	// Base query with joins
	baseQuery := `
		SELECT DISTINCT
			j.id,
			jd.title,
			jd.description,
			c.id as company_id,
			c.name as company_name,
			c.city as company_city,
			c.postal_code as company_postal_code,
			l.city as location_city,
			l.canton_code as location_canton,
			l.country_iso_code as location_country,
			l.postal_code as location_postal_code,
			CAST(l.lat AS FLOAT) as location_lat,
			CAST(l.lon AS FLOAT) as location_lon,
			e.permanent,
			e.immediately,
			e.workload_min,
			e.workload_max,
			e.start_date,
			j.external_url,
			j.source,
			j.status,
			j.created_time,
			j.updated_time
		FROM jobs j
		LEFT JOIN job_descriptions jd ON j.id = jd.job_id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN locations l ON j.location_id = l.id
		LEFT JOIN employment e ON j.id = e.job_id
	`

	// Default status filter
	conditions = append(conditions, "j.status = 'active'")

	// Text search
	if req.Query != "" {
		var searchTerms []string

		// Use AI to enhance query if enabled
		if req.UseAI && s.geminiClient != nil {
			aiResult, err := s.geminiClient.ParseNaturalLanguageQuery(ctx, req.Query)
			if err == nil && aiResult != nil {
				aiEnhanced = true
				// Use expanded query and keywords
				if aiResult.ExpandedQuery != "" {
					searchTerms = append(searchTerms, aiResult.ExpandedQuery)
				}
				searchTerms = append(searchTerms, aiResult.Keywords...)
				searchTerms = append(searchTerms, aiResult.Skills...)
				searchTerms = append(searchTerms, aiResult.JobTitles...)
			}
		}

		if len(searchTerms) == 0 {
			searchTerms = []string{req.Query}
		}

		// Build text search condition
		tsQuery := strings.Join(searchTerms, " | ")
		conditions = append(conditions, fmt.Sprintf(
			"(to_tsvector('german', COALESCE(jd.title, '') || ' ' || COALESCE(jd.description, '')) @@ plainto_tsquery('german', $%d) OR jd.title ILIKE $%d || '%%')",
			argNum, argNum+1,
		))
		args = append(args, tsQuery, req.Query)
		argNum += 2
	}

	// Apply filters
	if req.Filters != nil {
		// Location filters
		if req.Filters.Location != nil {
			loc := req.Filters.Location
			if loc.City != "" {
				conditions = append(conditions, fmt.Sprintf("l.city ILIKE $%d", argNum))
				args = append(args, "%"+loc.City+"%")
				argNum++
			}
			if loc.Canton != "" {
				conditions = append(conditions, fmt.Sprintf("l.canton_code = $%d", argNum))
				args = append(args, loc.Canton)
				argNum++
			}
			if loc.Country != "" {
				conditions = append(conditions, fmt.Sprintf("l.country_iso_code = $%d", argNum))
				args = append(args, loc.Country)
				argNum++
			}
			if len(loc.PostalCodes) > 0 {
				placeholders := make([]string, len(loc.PostalCodes))
				for i, pc := range loc.PostalCodes {
					placeholders[i] = fmt.Sprintf("$%d", argNum)
					args = append(args, pc)
					argNum++
				}
				conditions = append(conditions, fmt.Sprintf("l.postal_code IN (%s)", strings.Join(placeholders, ",")))
			}
			// Radius search using Haversine formula
			if loc.RadiusKM > 0 && loc.Lat != 0 && loc.Lon != 0 {
				conditions = append(conditions, fmt.Sprintf(`
					(6371 * acos(cos(radians($%d)) * cos(radians(CAST(l.lat AS FLOAT))) * cos(radians(CAST(l.lon AS FLOAT)) - radians($%d)) + sin(radians($%d)) * sin(radians(CAST(l.lat AS FLOAT))))) <= $%d`,
					argNum, argNum+1, argNum+2, argNum+3,
				))
				args = append(args, loc.Lat, loc.Lon, loc.Lat, loc.RadiusKM)
				argNum += 4
			}
		}

		// Workload filters
		if req.Filters.Workload != nil {
			if req.Filters.Workload.Min > 0 {
				conditions = append(conditions, fmt.Sprintf("CAST(e.workload_max AS INTEGER) >= $%d", argNum))
				args = append(args, req.Filters.Workload.Min)
				argNum++
			}
			if req.Filters.Workload.Max > 0 {
				conditions = append(conditions, fmt.Sprintf("CAST(e.workload_min AS INTEGER) <= $%d", argNum))
				args = append(args, req.Filters.Workload.Max)
				argNum++
			}
		}

		// Employment filters
		if req.Filters.Employment != nil {
			if req.Filters.Employment.Permanent != nil {
				conditions = append(conditions, fmt.Sprintf("e.permanent = $%d", argNum))
				args = append(args, *req.Filters.Employment.Permanent)
				argNum++
			}
			if req.Filters.Employment.Immediately != nil {
				conditions = append(conditions, fmt.Sprintf("e.immediately = $%d", argNum))
				args = append(args, *req.Filters.Employment.Immediately)
				argNum++
			}
		}

		// Company filters
		if req.Filters.Company != nil {
			if len(req.Filters.Company.Names) > 0 {
				placeholders := make([]string, len(req.Filters.Company.Names))
				for i, name := range req.Filters.Company.Names {
					placeholders[i] = fmt.Sprintf("$%d", argNum)
					args = append(args, name)
					argNum++
				}
				conditions = append(conditions, fmt.Sprintf("c.name IN (%s)", strings.Join(placeholders, ",")))
			}
			if len(req.Filters.Company.ExcludeNames) > 0 {
				placeholders := make([]string, len(req.Filters.Company.ExcludeNames))
				for i, name := range req.Filters.Company.ExcludeNames {
					placeholders[i] = fmt.Sprintf("$%d", argNum)
					args = append(args, name)
					argNum++
				}
				conditions = append(conditions, fmt.Sprintf("c.name NOT IN (%s)", strings.Join(placeholders, ",")))
			}
		}

		// Date filters
		if req.Filters.Date != nil {
			if req.Filters.Date.PostedAfter != "" {
				conditions = append(conditions, fmt.Sprintf("j.created_time >= $%d", argNum))
				args = append(args, req.Filters.Date.PostedAfter)
				argNum++
			}
			if req.Filters.Date.PostedBefore != "" {
				conditions = append(conditions, fmt.Sprintf("j.created_time <= $%d", argNum))
				args = append(args, req.Filters.Date.PostedBefore)
				argNum++
			}
		}

		// Language filter
		if req.Filters.Language != "" {
			conditions = append(conditions, fmt.Sprintf("jd.language_iso_code = $%d", argNum))
			args = append(args, req.Filters.Language)
			argNum++
		}

		// Source filter
		if req.Filters.Source != "" {
			conditions = append(conditions, fmt.Sprintf("j.source = $%d", argNum))
			args = append(args, req.Filters.Source)
			argNum++
		}

		// Status filter
		if req.Filters.Status != "" {
			// Override default status
			conditions = conditions[1:] // Remove default status condition
			conditions = append(conditions, fmt.Sprintf("j.status = $%d", argNum))
			args = append(args, req.Filters.Status)
			argNum++
		}
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Sorting
	orderBy := "j.created_time DESC"
	if req.Sort != nil && req.Sort.Field != "" {
		field := req.Sort.Field
		order := "DESC"
		if req.Sort.Order == "asc" {
			order = "ASC"
		}
		switch field {
		case "created_time":
			orderBy = "j.created_time " + order
		case "updated_time":
			orderBy = "j.updated_time " + order
		case "title":
			orderBy = "jd.title " + order
		case "company":
			orderBy = "c.name " + order
		}
	}

	// Full query
	fullQuery := fmt.Sprintf("%s %s ORDER BY %s LIMIT %d OFFSET %d",
		baseQuery, whereClause, orderBy, limit, offset)

	return fullQuery, args, aiEnhanced
}

// countResults counts total matching jobs.
func (s *Service) countResults(ctx context.Context, req *models.SearchRequest) (int, error) {
	// Simplified count query
	countQuery := `
		SELECT COUNT(DISTINCT j.id)
		FROM jobs j
		LEFT JOIN job_descriptions jd ON j.id = jd.job_id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN locations l ON j.location_id = l.id
		LEFT JOIN employment e ON j.id = e.job_id
		WHERE j.status = 'active'
	`

	if req.Query != "" {
		countQuery += fmt.Sprintf(" AND (jd.title ILIKE '%%%s%%' OR jd.description ILIKE '%%%s%%')", req.Query, req.Query)
	}

	var count int
	err := s.db.GetContext(ctx, &count, countQuery)
	return count, err
}

// scanJobResult scans a row into JobResult.
func (s *Service) scanJobResult(rows *sqlx.Rows) (*models.JobResult, error) {
	var job struct {
		ID                 string          `db:"id"`
		Title              sql.NullString  `db:"title"`
		Description        sql.NullString  `db:"description"`
		CompanyID          sql.NullInt64   `db:"company_id"`
		CompanyName        sql.NullString  `db:"company_name"`
		CompanyCity        sql.NullString  `db:"company_city"`
		CompanyPostalCode  sql.NullString  `db:"company_postal_code"`
		LocationCity       sql.NullString  `db:"location_city"`
		LocationCanton     sql.NullString  `db:"location_canton"`
		LocationCountry    sql.NullString  `db:"location_country"`
		LocationPostalCode sql.NullString  `db:"location_postal_code"`
		LocationLat        sql.NullFloat64 `db:"location_lat"`
		LocationLon        sql.NullFloat64 `db:"location_lon"`
		Permanent          sql.NullBool    `db:"permanent"`
		Immediately        sql.NullBool    `db:"immediately"`
		WorkloadMin        sql.NullString  `db:"workload_min"`
		WorkloadMax        sql.NullString  `db:"workload_max"`
		StartDate          sql.NullString  `db:"start_date"`
		ExternalURL        sql.NullString  `db:"external_url"`
		Source             string          `db:"source"`
		Status             string          `db:"status"`
		CreatedTime        time.Time       `db:"created_time"`
		UpdatedTime        time.Time       `db:"updated_time"`
	}

	if err := rows.StructScan(&job); err != nil {
		return nil, err
	}

	result := &models.JobResult{
		ID:          job.ID,
		Title:       job.Title.String,
		Description: truncate(job.Description.String, 300),
		Source:      job.Source,
		Status:      job.Status,
		CreatedTime: job.CreatedTime,
		UpdatedTime: job.UpdatedTime,
		ExternalURL: job.ExternalURL.String,
	}

	if job.CompanyName.Valid {
		result.Company = &models.CompanyInfo{
			ID:         job.CompanyID.Int64,
			Name:       job.CompanyName.String,
			City:       job.CompanyCity.String,
			PostalCode: job.CompanyPostalCode.String,
		}
	}

	if job.LocationCity.Valid {
		result.Location = &models.LocationInfo{
			City:       job.LocationCity.String,
			Canton:     job.LocationCanton.String,
			Country:    job.LocationCountry.String,
			PostalCode: job.LocationPostalCode.String,
			Lat:        job.LocationLat.Float64,
			Lon:        job.LocationLon.Float64,
		}
	}

	result.Employment = &models.EmploymentInfo{
		Permanent:   job.Permanent.Bool,
		Immediately: job.Immediately.Bool,
		WorkloadMin: job.WorkloadMin.String,
		WorkloadMax: job.WorkloadMax.String,
		StartDate:   job.StartDate.String,
	}

	return result, nil
}

// GetJob retrieves a single job by ID.
func (s *Service) GetJob(ctx context.Context, id string) (*models.JobResult, error) {
	query := `
		SELECT
			j.id,
			jd.title,
			jd.description,
			c.id as company_id,
			c.name as company_name,
			c.city as company_city,
			c.postal_code as company_postal_code,
			l.city as location_city,
			l.canton_code as location_canton,
			l.country_iso_code as location_country,
			l.postal_code as location_postal_code,
			CAST(l.lat AS FLOAT) as location_lat,
			CAST(l.lon AS FLOAT) as location_lon,
			e.permanent,
			e.immediately,
			e.workload_min,
			e.workload_max,
			e.start_date,
			j.external_url,
			j.source,
			j.status,
			j.created_time,
			j.updated_time
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
		return s.scanJobResult(rows)
	}

	return nil, nil
}

// GetFilterOptions returns available filter options.
func (s *Service) GetFilterOptions(ctx context.Context) (*models.FilterOptions, error) {
	options := &models.FilterOptions{
		Languages: []string{"de", "fr", "it", "en"},
		Sources:   []string{"jobroom", "platform"},
	}

	// Get distinct cantons
	var cantons []string
	s.db.SelectContext(ctx, &cantons, "SELECT DISTINCT canton_code FROM locations WHERE canton_code != '' ORDER BY canton_code LIMIT 30")
	options.Cantons = cantons

	// Get distinct cities
	var cities []string
	s.db.SelectContext(ctx, &cities, "SELECT DISTINCT city FROM locations WHERE city != '' ORDER BY city LIMIT 100")
	options.Cities = cities

	// Get top companies
	var companies []string
	s.db.SelectContext(ctx, &companies, "SELECT DISTINCT name FROM companies WHERE name != '' ORDER BY name LIMIT 100")
	options.Companies = companies

	return options, nil
}

// SaveSearch saves a search for a user.
func (s *Service) SaveSearch(ctx context.Context, userID uuid.UUID, name string, req *models.SearchRequest) (*models.SavedSearch, error) {
	filtersJSON, _ := json.Marshal(req.Filters)

	var saved models.SavedSearch
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO saved_searches (user_id, name, query, filters)
		VALUES ($1, $2, $3, $4)
		RETURNING *`,
		userID, name, req.Query, string(filtersJSON),
	).StructScan(&saved)

	if err != nil {
		return nil, fmt.Errorf("failed to save search: %w", err)
	}

	return &saved, nil
}

// GetSavedSearches returns saved searches for a user.
func (s *Service) GetSavedSearches(ctx context.Context, userID uuid.UUID) ([]models.SavedSearch, error) {
	var searches []models.SavedSearch
	err := s.db.SelectContext(ctx, &searches, `
		SELECT * FROM saved_searches WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	return searches, err
}

// DeleteSavedSearch deletes a saved search.
func (s *Service) DeleteSavedSearch(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM saved_searches WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
