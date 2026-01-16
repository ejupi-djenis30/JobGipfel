package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// SearchRequest is the main search request structure.
type SearchRequest struct {
	Query      string           `json:"query"` // Natural language or keyword query
	Filters    *SearchFilters   `json:"filters,omitempty"`
	Sort       *SortOptions     `json:"sort,omitempty"`
	Pagination *PaginationInput `json:"pagination,omitempty"`
	UseAI      bool             `json:"use_ai"` // Use AI for semantic search
}

// SearchFilters contains all available filters.
type SearchFilters struct {
	Location   *LocationFilter   `json:"location,omitempty"`
	Workload   *WorkloadFilter   `json:"workload,omitempty"`
	Employment *EmploymentFilter `json:"employment,omitempty"`
	Company    *CompanyFilter    `json:"company,omitempty"`
	Date       *DateFilter       `json:"date,omitempty"`
	Language   string            `json:"language,omitempty"` // de, fr, it, en
	Source     string            `json:"source,omitempty"`   // jobroom, platform
	Status     string            `json:"status,omitempty"`   // active, inactive, expired
}

// LocationFilter for geographic filtering.
type LocationFilter struct {
	City        string   `json:"city,omitempty"`
	Canton      string   `json:"canton,omitempty"`
	Country     string   `json:"country,omitempty"`
	PostalCodes []string `json:"postal_codes,omitempty"`
	RadiusKM    int      `json:"radius_km,omitempty"`
	Lat         float64  `json:"lat,omitempty"`
	Lon         float64  `json:"lon,omitempty"`
}

// WorkloadFilter for workload percentage.
type WorkloadFilter struct {
	Min int `json:"min,omitempty"` // 0-100
	Max int `json:"max,omitempty"` // 0-100
}

// EmploymentFilter for employment terms.
type EmploymentFilter struct {
	Permanent   *bool `json:"permanent,omitempty"`
	Immediately *bool `json:"immediately,omitempty"`
	Temporary   *bool `json:"temporary,omitempty"` // Short employment
}

// CompanyFilter for company filtering.
type CompanyFilter struct {
	Names            []string `json:"names,omitempty"` // Include companies
	ExcludeNames     []string `json:"exclude_names,omitempty"`
	ExcludeAnonymous bool     `json:"exclude_anonymous,omitempty"`
}

// DateFilter for date range filtering.
type DateFilter struct {
	PostedAfter  string `json:"posted_after,omitempty"`  // ISO date
	PostedBefore string `json:"posted_before,omitempty"` // ISO date
}

// SortOptions for search result sorting.
type SortOptions struct {
	Field string `json:"field"` // created_time, updated_time, relevance
	Order string `json:"order"` // asc, desc
}

// PaginationInput for pagination.
type PaginationInput struct {
	Page    int `json:"page"`     // 1-indexed
	PerPage int `json:"per_page"` // Items per page
}

// SearchResponse is the search result.
type SearchResponse struct {
	Jobs       []JobResult `json:"jobs"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	Query      string      `json:"query,omitempty"`
	AIEnhanced bool        `json:"ai_enhanced"`
}

// JobResult is a job in search results.
type JobResult struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Company     *CompanyInfo    `json:"company,omitempty"`
	Location    *LocationInfo   `json:"location,omitempty"`
	Employment  *EmploymentInfo `json:"employment,omitempty"`
	ExternalURL string          `json:"external_url,omitempty"`
	Source      string          `json:"source"`
	Status      string          `json:"status"`
	CreatedTime time.Time       `json:"created_time"`
	UpdatedTime time.Time       `json:"updated_time"`
	Relevance   float64         `json:"relevance,omitempty"` // AI similarity score
}

// CompanyInfo for search results.
type CompanyInfo struct {
	ID         int64  `json:"id,omitempty"`
	Name       string `json:"name"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}

// LocationInfo for search results.
type LocationInfo struct {
	City       string  `json:"city"`
	Canton     string  `json:"canton,omitempty"`
	Country    string  `json:"country,omitempty"`
	PostalCode string  `json:"postal_code,omitempty"`
	Lat        float64 `json:"lat,omitempty"`
	Lon        float64 `json:"lon,omitempty"`
}

// EmploymentInfo for search results.
type EmploymentInfo struct {
	Permanent   bool   `json:"permanent"`
	Immediately bool   `json:"immediately"`
	WorkloadMin string `json:"workload_min,omitempty"`
	WorkloadMax string `json:"workload_max,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
}

// FilterOptions returns available filter options.
type FilterOptions struct {
	Cantons   []string `json:"cantons"`
	Cities    []string `json:"cities"`
	Companies []string `json:"companies"`
	Languages []string `json:"languages"`
	Sources   []string `json:"sources"`
}

// SavedSearch represents a saved search.
type SavedSearch struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	UserID    uuid.UUID      `json:"user_id" db:"user_id"`
	Name      string         `json:"name" db:"name"`
	Query     string         `json:"query" db:"query"`
	Filters   sql.NullString `json:"filters" db:"filters"` // JSON
	NotifyNew bool           `json:"notify_new" db:"notify_new"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// RecommendRequest for personalized recommendations.
type RecommendRequest struct {
	Limit int `json:"limit,omitempty"`
}

// AIQueryResult from Gemini.
type AIQueryResult struct {
	Keywords      []string `json:"keywords"`
	Skills        []string `json:"skills"`
	JobTitles     []string `json:"job_titles"`
	Locations     []string `json:"locations"`
	Intent        string   `json:"intent"` // search, filter, recommendation
	ExpandedQuery string   `json:"expanded_query"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
