package models

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ScrapeRequest contains all filtering options for job scraping.
type ScrapeRequest struct {
	// Strategy: "full" scrapes all pages, "incremental" stops when existing jobs found
	Strategy string `json:"strategy"`

	// MaxPages is the maximum number of pages to scrape (0 = unlimited)
	MaxPages int `json:"max_pages"`

	// StartPage is the page to start scraping from (0 = first page)
	StartPage int `json:"start_page"`

	// Keywords for full-text search
	Keywords string `json:"keywords"`

	// Cantons filters by canton codes (e.g., ["ZH", "BE", "GE"])
	Cantons []string `json:"cantons"`

	// WorkloadMin is the minimum workload percentage (0-100)
	WorkloadMin int `json:"workload_min"`

	// WorkloadMax is the maximum workload percentage (0-100)
	WorkloadMax int `json:"workload_max"`

	// Permanent filters by contract type: nil=both, true=permanent, false=temporary
	Permanent *bool `json:"permanent"`

	// DaysBack filters jobs published within the last N days
	DaysBack int `json:"days_back"`

	// Polite enables random delays between requests
	Polite bool `json:"polite"`
}

// ToJSON converts the request to JSON for storage.
func (r *ScrapeRequest) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildQuery constructs URL query parameters from the ScrapeRequest.
func (r *ScrapeRequest) BuildQuery(page int) url.Values {
	q := url.Values{}

	// Pagination
	q.Set("page", strconv.Itoa(page))
	q.Set("size", "20")
	q.Set("sort", "date_desc")
	q.Set("_ng", "ZW4=")

	// Keywords
	if r.Keywords != "" {
		q.Set("keywords", r.Keywords)
	}

	// Canton codes (multiple values)
	for _, canton := range r.Cantons {
		q.Add("cantonCode", canton)
	}

	// Workload range
	if r.WorkloadMin > 0 {
		q.Set("workloadPercentageMin", strconv.Itoa(r.WorkloadMin))
	}
	if r.WorkloadMax > 0 && r.WorkloadMax <= 100 {
		q.Set("workloadPercentageMax", strconv.Itoa(r.WorkloadMax))
	}

	// Contract type
	if r.Permanent != nil {
		q.Set("permanent", strconv.FormatBool(*r.Permanent))
	}

	// Publication date filter (calculate from DaysBack)
	if r.DaysBack > 0 {
		fromDate := time.Now().AddDate(0, 0, -r.DaysBack).Format("2006-01-02")
		q.Set("publicationStartDateFrom", fromDate)
	}

	return q
}

// DefaultScrapeRequest returns a ScrapeRequest with sensible defaults.
func DefaultScrapeRequest() ScrapeRequest {
	return ScrapeRequest{
		Strategy:    "full",
		MaxPages:    5,
		WorkloadMin: 10,
		WorkloadMax: 100,
		DaysBack:    60,
		Polite:      true,
	}
}

// SearchRequestBody is the JSON body for the job-room.ch search API.
type SearchRequestBody struct {
	WorkloadPercentageMin int      `json:"workloadPercentageMin"`
	WorkloadPercentageMax int      `json:"workloadPercentageMax"`
	Permanent             *bool    `json:"permanent"`
	CompanyName           *string  `json:"companyName"`
	OnlineSince           int      `json:"onlineSince"`
	DisplayRestricted     bool     `json:"displayRestricted"`
	ProfessionCodes       []string `json:"professionCodes"`
	Keywords              []string `json:"keywords"`
	CommunalCodes         []string `json:"communalCodes"`
	CantonCodes           []string `json:"cantonCodes"`
}

// BuildSearchBody builds the JSON request body for the search API.
func (r *ScrapeRequest) BuildSearchBody() SearchRequestBody {
	workloadMin := r.WorkloadMin
	if workloadMin <= 0 {
		workloadMin = 10
	}
	workloadMax := r.WorkloadMax
	if workloadMax <= 0 || workloadMax > 100 {
		workloadMax = 100
	}

	// Calculate onlineSince from DaysBack
	onlineSince := 60 // Default to 60 days
	if r.DaysBack > 0 {
		onlineSince = r.DaysBack
	}

	// Parse keywords into array
	var keywords []string
	if r.Keywords != "" {
		keywords = strings.Fields(r.Keywords)
	} else {
		keywords = []string{}
	}

	// Ensure cantons is not nil
	cantonCodes := r.Cantons
	if cantonCodes == nil {
		cantonCodes = []string{}
	}

	return SearchRequestBody{
		WorkloadPercentageMin: workloadMin,
		WorkloadPercentageMax: workloadMax,
		Permanent:             r.Permanent,
		CompanyName:           nil,
		OnlineSince:           onlineSince,
		DisplayRestricted:     false,
		ProfessionCodes:       []string{},
		Keywords:              keywords,
		CommunalCodes:         []string{},
		CantonCodes:           cantonCodes,
	}
}

// AllSwissCantons returns all 26 Swiss canton codes.
func AllSwissCantons() []string {
	return []string{
		"AG", "AI", "AR", "BE", "BL", "BS", "FR", "GE", "GL", "GR",
		"JU", "LU", "NE", "NW", "OW", "SG", "SH", "SO", "SZ", "TG",
		"TI", "UR", "VD", "VS", "ZG", "ZH",
	}
}

// ValidateCanton checks if a canton code is valid.
func ValidateCanton(code string) bool {
	code = strings.ToUpper(code)
	for _, c := range AllSwissCantons() {
		if c == code {
			return true
		}
	}
	return false
}
