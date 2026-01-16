package models

import (
	"time"

	"github.com/google/uuid"
)

// MatchedJob represents a job with matching score.
type MatchedJob struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Company     *CompanyInfo    `json:"company,omitempty"`
	Location    *LocationInfo   `json:"location,omitempty"`
	Employment  *EmploymentInfo `json:"employment,omitempty"`
	ExternalURL string          `json:"external_url,omitempty"`
	Source      string          `json:"source"`
	CreatedTime time.Time       `json:"created_time"`

	// Matching scores
	MatchScore      float64  `json:"match_score"`      // Overall 0-100
	SkillsMatch     float64  `json:"skills_match"`     // 0-100
	ExperienceMatch float64  `json:"experience_match"` // 0-100
	LocationMatch   float64  `json:"location_match"`   // 0-100
	WorkloadMatch   float64  `json:"workload_match"`   // 0-100
	MatchedSkills   []string `json:"matched_skills"`
	MissingSkills   []string `json:"missing_skills"`
	MatchReasons    []string `json:"match_reasons"`
}

// CompanyInfo for matched jobs.
type CompanyInfo struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name"`
	City string `json:"city,omitempty"`
}

// LocationInfo for matched jobs.
type LocationInfo struct {
	City    string  `json:"city"`
	Canton  string  `json:"canton,omitempty"`
	Country string  `json:"country,omitempty"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

// EmploymentInfo for matched jobs.
type EmploymentInfo struct {
	Permanent   bool   `json:"permanent"`
	Immediately bool   `json:"immediately"`
	WorkloadMin string `json:"workload_min,omitempty"`
	WorkloadMax string `json:"workload_max,omitempty"`
}

// MatchRequest for getting matches.
type MatchRequest struct {
	Limit          int      `json:"limit,omitempty"`
	MinScore       float64  `json:"min_score,omitempty"`       // Minimum match score
	PrioritySkills []string `json:"priority_skills,omitempty"` // Skills to prioritize
	ExcludeApplied bool     `json:"exclude_applied,omitempty"` // Exclude jobs already applied to
	Location       string   `json:"location,omitempty"`        // Preferred location
	RemoteOnly     bool     `json:"remote_only,omitempty"`
}

// MatchResponse contains matched jobs.
type MatchResponse struct {
	Jobs      []MatchedJob `json:"jobs"`
	Total     int          `json:"total"`
	ProfileID uuid.UUID    `json:"profile_id"`
	MatchedAt time.Time    `json:"matched_at"`
}

// JobScoreRequest for scoring a specific job.
type JobScoreRequest struct {
	JobID string `uri:"job_id" binding:"required"`
}

// JobScoreResponse for a specific job score.
type JobScoreResponse struct {
	JobID           string   `json:"job_id"`
	JobTitle        string   `json:"job_title"`
	MatchScore      float64  `json:"match_score"`
	SkillsMatch     float64  `json:"skills_match"`
	ExperienceMatch float64  `json:"experience_match"`
	LocationMatch   float64  `json:"location_match"`
	WorkloadMatch   float64  `json:"workload_match"`
	MatchedSkills   []string `json:"matched_skills"`
	MissingSkills   []string `json:"missing_skills"`
	Strengths       []string `json:"strengths"`
	Gaps            []string `json:"gaps"`
	Recommendations []string `json:"recommendations"`
}

// ProfileAnalysis from AI.
type ProfileAnalysis struct {
	SkillsExtracted []string `json:"skills_extracted"`
	ExperienceYears int      `json:"experience_years"`
	SeniorityLevel  string   `json:"seniority_level"` // junior, mid, senior
	Industries      []string `json:"industries"`
	SuggestedSkills []string `json:"suggested_skills"`
	ProfileStrength int      `json:"profile_strength"` // 0-100
	ImprovementTips []string `json:"improvement_tips"`
}

// UserProfile from auth_service.
type UserProfile struct {
	UserID      uuid.UUID    `json:"user_id"`
	FirstName   string       `json:"first_name"`
	LastName    string       `json:"last_name"`
	Headline    string       `json:"headline"`
	Summary     string       `json:"summary"`
	City        string       `json:"city"`
	Country     string       `json:"country"`
	Skills      []Skill      `json:"skills"`
	Experiences []Experience `json:"experiences"`
	Education   []Education  `json:"education"`
}

// Skill from profile.
type Skill struct {
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	Level    string `json:"level,omitempty"` // beginner, intermediate, expert
}

// Experience from profile.
type Experience struct {
	Title       string  `json:"title"`
	CompanyName string  `json:"company_name"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	IsCurrent   bool    `json:"is_current"`
	Description string  `json:"description"`
}

// Education from profile.
type Education struct {
	InstitutionName string `json:"institution_name"`
	Degree          string `json:"degree"`
	FieldOfStudy    string `json:"field_of_study"`
}

// ErrorResponse standard error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
