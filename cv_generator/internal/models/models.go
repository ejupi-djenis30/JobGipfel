package models

import (
	"time"
)

// CVStyle represents available CV styles.
type CVStyle string

const (
	StyleModern     CVStyle = "modern"
	StyleMinimalist CVStyle = "minimalist"
	StyleClassic    CVStyle = "classic"
	StyleCreative   CVStyle = "creative"
)

// ColorScheme represents available color schemes.
type ColorScheme string

const (
	ColorBlue    ColorScheme = "blue"
	ColorGreen   ColorScheme = "green"
	ColorDark    ColorScheme = "dark"
	ColorNeutral ColorScheme = "neutral"
	ColorPurple  ColorScheme = "purple"
	ColorRed     ColorScheme = "red"
)

// CVSections controls which sections to include.
type CVSections struct {
	Summary        bool `json:"summary"`
	Experiences    bool `json:"experiences"`
	Education      bool `json:"education"`
	Skills         bool `json:"skills"`
	Certifications bool `json:"certifications"`
}

// GenerateCVRequest is the request to generate a CV.
type GenerateCVRequest struct {
	Style              CVStyle     `json:"style"`
	IncludePhoto       bool        `json:"include_photo"`
	ColorScheme        ColorScheme `json:"color_scheme"`
	Sections           CVSections  `json:"sections"`
	MaxExperiences     int         `json:"max_experiences"`
	MaxEducation       int         `json:"max_education"`
	MaxSkills          int         `json:"max_skills"`
	Language           string      `json:"language"`
	CustomInstructions string      `json:"custom_instructions"`
}

// DefaultCVRequest returns sensible defaults.
func DefaultCVRequest() GenerateCVRequest {
	return GenerateCVRequest{
		Style:        StyleModern,
		IncludePhoto: true,
		ColorScheme:  ColorBlue,
		Sections: CVSections{
			Summary:        true,
			Experiences:    true,
			Education:      true,
			Skills:         true,
			Certifications: true,
		},
		MaxExperiences: 5,
		MaxEducation:   3,
		MaxSkills:      15,
		Language:       "en",
	}
}

// CVOptions describes available customization options.
type CVOptions struct {
	Styles       []CVStyle     `json:"styles"`
	ColorSchemes []ColorScheme `json:"color_schemes"`
	Languages    []string      `json:"languages"`
}

// GetAvailableOptions returns all available customization options.
func GetAvailableOptions() CVOptions {
	return CVOptions{
		Styles:       []CVStyle{StyleModern, StyleMinimalist, StyleClassic, StyleCreative},
		ColorSchemes: []ColorScheme{ColorBlue, ColorGreen, ColorDark, ColorNeutral, ColorPurple, ColorRed},
		Languages:    []string{"en", "de", "fr", "it", "es"},
	}
}

// ==================== Resume Data (from auth_service) ====================

// ResumeData is the complete profile data from auth_service.
type ResumeData struct {
	User        UserInfo     `json:"user"`
	Profile     *Profile     `json:"profile"`
	Experiences []Experience `json:"experiences"`
	Education   []Education  `json:"education"`
	Skills      []Skill      `json:"skills"`
}

// UserInfo from auth_service.
type UserInfo struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// Profile from auth_service.
type Profile struct {
	FirstName            *string  `json:"first_name"`
	LastName             *string  `json:"last_name"`
	Headline             *string  `json:"headline"`
	Summary              *string  `json:"summary"`
	Phone                *string  `json:"phone"`
	Website              *string  `json:"website"`
	LinkedInURL          *string  `json:"linkedin_url"`
	GithubURL            *string  `json:"github_url"`
	City                 *string  `json:"city"`
	Country              *string  `json:"country"`
	PreferredJobTitles   []string `json:"preferred_job_titles"`
	SalaryExpectationMin *int     `json:"salary_expectation_min"`
	SalaryExpectationMax *int     `json:"salary_expectation_max"`
	SalaryCurrency       *string  `json:"salary_currency"`
}

// Experience from auth_service.
type Experience struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	CompanyName    string   `json:"company_name"`
	EmploymentType *string  `json:"employment_type"`
	Location       *string  `json:"location"`
	StartDate      string   `json:"start_date"`
	EndDate        *string  `json:"end_date"`
	IsCurrent      bool     `json:"is_current"`
	Description    *string  `json:"description"`
	Achievements   []string `json:"achievements"`
	SkillsUsed     []string `json:"skills_used"`
}

// Education from auth_service.
type Education struct {
	ID              string   `json:"id"`
	InstitutionName string   `json:"institution_name"`
	Degree          *string  `json:"degree"`
	FieldOfStudy    *string  `json:"field_of_study"`
	Grade           *string  `json:"grade"`
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	IsCurrent       bool     `json:"is_current"`
	Description     *string  `json:"description"`
	Activities      []string `json:"activities"`
}

// Skill from auth_service.
type Skill struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Category         *string `json:"category"`
	ProficiencyLevel *string `json:"proficiency_level"`
	YearsExperience  *int    `json:"years_of_experience"`
	IsCertification  bool    `json:"is_certification"`
}

// GenerateCVResponse is the response after generating a CV.
type GenerateCVResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	HTML     string `json:"html,omitempty"`
	PDFBytes []byte `json:"-"` // Not in JSON, sent as file
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
