package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ApplicationMethod represents how the job was applied to.
type ApplicationMethod string

const (
	MethodEmail ApplicationMethod = "email"
	MethodWeb   ApplicationMethod = "web"
)

// ApplicationStatus represents the status of an application.
type ApplicationStatus string

const (
	StatusPending    ApplicationStatus = "pending"
	StatusInProgress ApplicationStatus = "in_progress"
	StatusSent       ApplicationStatus = "sent"
	StatusFailed     ApplicationStatus = "failed"
	StatusRetrying   ApplicationStatus = "retrying"
)

// Application represents a job application.
type Application struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	UserID         uuid.UUID         `json:"user_id" db:"user_id"`
	JobID          sql.NullString    `json:"job_id" db:"job_id"` // Reference to jobs table if applicable
	JobTitle       string            `json:"job_title" db:"job_title"`
	CompanyName    string            `json:"company_name" db:"company_name"`
	JobURL         sql.NullString    `json:"job_url" db:"job_url"`
	Method         ApplicationMethod `json:"method" db:"method"`
	Status         ApplicationStatus `json:"status" db:"status"`
	RecipientEmail sql.NullString    `json:"recipient_email" db:"recipient_email"` // For email applications
	CoverLetter    sql.NullString    `json:"cover_letter" db:"cover_letter"`
	CVData         sql.NullString    `json:"-" db:"cv_data"` // Base64 encoded PDF (not in JSON response)
	CVStyle        sql.NullString    `json:"cv_style" db:"cv_style"`
	CVColorScheme  sql.NullString    `json:"cv_color_scheme" db:"cv_color_scheme"`
	ScreenshotURL  sql.NullString    `json:"screenshot_url" db:"screenshot_url"` // For web applications
	ErrorMessage   sql.NullString    `json:"error_message" db:"error_message"`
	RetryCount     int               `json:"retry_count" db:"retry_count"`
	SentAt         sql.NullTime      `json:"sent_at" db:"sent_at"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

// CVOptions for application requests
type CVOptions struct {
	Style       string `json:"cv_style"`        // modern, minimalist, classic, creative
	ColorScheme string `json:"cv_color_scheme"` // blue, green, dark, neutral, purple, red
}

// EmailApplicationRequest is the request to apply via email.
type EmailApplicationRequest struct {
	JobTitle       string    `json:"job_title" binding:"required"`
	CompanyName    string    `json:"company_name" binding:"required"`
	RecipientEmail string    `json:"recipient_email" binding:"required,email"`
	JobURL         string    `json:"job_url"`
	JobDescription string    `json:"job_description"` // For cover letter generation
	JobID          string    `json:"job_id"`          // Optional reference
	UseUserGmail   bool      `json:"use_user_gmail"`  // Use user's Gmail or platform SMTP
	CustomMessage  string    `json:"custom_message"`  // Additional instructions for cover letter
	CVOptions      CVOptions `json:"cv_options"`      // CV generation options
}

// WebApplicationRequest is the request to apply via web form.
type WebApplicationRequest struct {
	JobTitle       string    `json:"job_title" binding:"required"`
	CompanyName    string    `json:"company_name" binding:"required"`
	JobURL         string    `json:"job_url" binding:"required,url"`
	JobDescription string    `json:"job_description"` // For form filling context
	JobID          string    `json:"job_id"`          // Optional reference
	CustomMessage  string    `json:"custom_message"`  // Additional instructions
	CVOptions      CVOptions `json:"cv_options"`      // CV generation options
}

// CoverLetterRequest is the request to generate a cover letter.
type CoverLetterRequest struct {
	JobTitle       string `json:"job_title" binding:"required"`
	CompanyName    string `json:"company_name" binding:"required"`
	JobDescription string `json:"job_description" binding:"required"`
	CustomMessage  string `json:"custom_message"`
	Language       string `json:"language"`
}

// CoverLetterResponse is the response with generated cover letter.
type CoverLetterResponse struct {
	CoverLetter string `json:"cover_letter"`
	Subject     string `json:"subject"`
}

// ApplicationResponse is the API response for an application.
type ApplicationResponse struct {
	ID          uuid.UUID         `json:"id"`
	JobTitle    string            `json:"job_title"`
	CompanyName string            `json:"company_name"`
	Method      ApplicationMethod `json:"method"`
	Status      ApplicationStatus `json:"status"`
	SentAt      *time.Time        `json:"sent_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ResumeData from auth_service (simplified).
type ResumeData struct {
	User        UserInfo     `json:"user"`
	Profile     *Profile     `json:"profile"`
	Experiences []Experience `json:"experiences"`
	Education   []Education  `json:"education"`
	Skills      []Skill      `json:"skills"`
}

type UserInfo struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

type Profile struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Headline    *string `json:"headline"`
	Summary     *string `json:"summary"`
	Phone       *string `json:"phone"`
	City        *string `json:"city"`
	Country     *string `json:"country"`
	LinkedInURL *string `json:"linkedin_url"`
	GithubURL   *string `json:"github_url"`
	Website     *string `json:"website"`
}

type Experience struct {
	Title        string   `json:"title"`
	CompanyName  string   `json:"company_name"`
	StartDate    string   `json:"start_date"`
	EndDate      *string  `json:"end_date"`
	IsCurrent    bool     `json:"is_current"`
	Description  *string  `json:"description"`
	Achievements []string `json:"achievements"`
}

type Education struct {
	InstitutionName string  `json:"institution_name"`
	Degree          *string `json:"degree"`
	FieldOfStudy    *string `json:"field_of_study"`
}

type Skill struct {
	Name     string  `json:"name"`
	Category *string `json:"category"`
}

// FormField represents a form field detected on a web page.
type FormField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // text, email, tel, textarea, select, file
	Label       string   `json:"label"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder"`
	Options     []string `json:"options"` // For select fields
}

// FormResponse is the AI-generated response for a form field.
type FormResponse struct {
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
