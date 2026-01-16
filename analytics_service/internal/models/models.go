package models

import (
	"time"

	"github.com/google/uuid"
)

// DashboardResponse is the main dashboard data.
type DashboardResponse struct {
	Applications    ApplicationStats `json:"applications"`
	Market          MarketStats      `json:"market"`
	Activity        []ActivityItem   `json:"activity"`
	Recommendations ProfileInsights  `json:"recommendations"`
	GeneratedAt     time.Time        `json:"generated_at"`
}

// ApplicationStats for user's applications.
type ApplicationStats struct {
	Total           int             `json:"total"`
	Sent            int             `json:"sent"`
	Pending         int             `json:"pending"`
	Failed          int             `json:"failed"`
	InProgress      int             `json:"in_progress"`
	ByMethod        MethodBreakdown `json:"by_method"`
	ResponseRate    float64         `json:"response_rate"`
	AvgResponseDays float64         `json:"avg_response_days"`
	ThisWeek        int             `json:"this_week"`
	ThisMonth       int             `json:"this_month"`
}

// MethodBreakdown by application method.
type MethodBreakdown struct {
	Email int `json:"email"`
	Web   int `json:"web"`
}

// MarketStats for job market insights.
type MarketStats struct {
	TotalJobs      int            `json:"total_jobs"`
	ActiveJobs     int            `json:"active_jobs"`
	NewToday       int            `json:"new_today"`
	NewThisWeek    int            `json:"new_this_week"`
	TopLocations   []LocationStat `json:"top_locations"`
	TopCompanies   []CompanyStat  `json:"top_companies"`
	TopSkills      []SkillStat    `json:"top_skills"`
	WorkloadTrends WorkloadStats  `json:"workload_trends"`
}

// LocationStat for location statistics.
type LocationStat struct {
	City     string `json:"city"`
	Canton   string `json:"canton,omitempty"`
	JobCount int    `json:"job_count"`
}

// CompanyStat for company statistics.
type CompanyStat struct {
	Name     string `json:"name"`
	JobCount int    `json:"job_count"`
}

// SkillStat for skill demand.
type SkillStat struct {
	Skill  string  `json:"skill"`
	Count  int     `json:"count"`
	Trend  string  `json:"trend"` // up, down, stable
	Growth float64 `json:"growth_percent,omitempty"`
}

// WorkloadStats for workload trends.
type WorkloadStats struct {
	FullTime int `json:"full_time"` // 80-100%
	PartTime int `json:"part_time"` // 40-79%
	Minimal  int `json:"minimal"`   // <40%
}

// ActivityItem for user activity timeline.
type ActivityItem struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"` // application, search, profile_update
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ProfileInsights for profile recommendations.
type ProfileInsights struct {
	ProfileCompleteness int      `json:"profile_completeness"` // 0-100
	SuggestedSkills     []string `json:"suggested_skills"`
	RecommendedActions  []string `json:"recommended_actions"`
	MatchingJobsCount   int      `json:"matching_jobs_count"`
}

// TimeSeriesPoint for charts.
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// ApplicationTrend for application trends over time.
type ApplicationTrend struct {
	Period string            `json:"period"` // daily, weekly, monthly
	Data   []TimeSeriesPoint `json:"data"`
}

// StatsRequest for filtering stats.
type StatsRequest struct {
	Period    string `form:"period"` // 7d, 30d, 90d, all
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

// ErrorResponse standard error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
