package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents an authenticated user.
type User struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	Email         string         `json:"email" db:"email"`
	EmailVerified bool           `json:"email_verified" db:"email_verified"`
	GoogleID      sql.NullString `json:"-" db:"google_id"`
	LinkedInID    sql.NullString `json:"-" db:"linkedin_id"`
	AvatarURL     sql.NullString `json:"avatar_url" db:"avatar_url"`
	Status        string         `json:"status" db:"status"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
	LastLoginAt   sql.NullTime   `json:"last_login_at" db:"last_login_at"`
}

// UserResponse is the API response for a user.
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

// ToResponse converts User to UserResponse.
func (u *User) ToResponse() UserResponse {
	resp := UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Status:        u.Status,
		CreatedAt:     u.CreatedAt,
	}
	if u.AvatarURL.Valid {
		resp.AvatarURL = &u.AvatarURL.String
	}
	if u.LastLoginAt.Valid {
		resp.LastLoginAt = &u.LastLoginAt.Time
	}
	return resp
}

// Profile represents user's career profile.
type Profile struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	UserID               uuid.UUID       `json:"user_id" db:"user_id"`
	FirstName            sql.NullString  `json:"first_name" db:"first_name"`
	LastName             sql.NullString  `json:"last_name" db:"last_name"`
	Headline             sql.NullString  `json:"headline" db:"headline"`
	Summary              sql.NullString  `json:"summary" db:"summary"`
	Phone                sql.NullString  `json:"phone" db:"phone"`
	Website              sql.NullString  `json:"website" db:"website"`
	LinkedInURL          sql.NullString  `json:"linkedin_url" db:"linkedin_url"`
	GithubURL            sql.NullString  `json:"github_url" db:"github_url"`
	City                 sql.NullString  `json:"city" db:"city"`
	Country              sql.NullString  `json:"country" db:"country"`
	PostalCode           sql.NullString  `json:"postal_code" db:"postal_code"`
	PreferredJobTitles   pq.StringArray  `json:"preferred_job_titles" db:"preferred_job_titles"`
	PreferredLocations   pq.StringArray  `json:"preferred_locations" db:"preferred_locations"`
	SalaryExpectationMin sql.NullInt32   `json:"salary_expectation_min" db:"salary_expectation_min"`
	SalaryExpectationMax sql.NullInt32   `json:"salary_expectation_max" db:"salary_expectation_max"`
	SalaryCurrency       sql.NullString  `json:"salary_currency" db:"salary_currency"`
	WorkAuthorization    sql.NullString  `json:"work_authorization" db:"work_authorization"`
	WillingToRelocate    bool            `json:"willing_to_relocate" db:"willing_to_relocate"`
	ImportedFrom         sql.NullString  `json:"imported_from" db:"imported_from"`
	RawImportData        json.RawMessage `json:"-" db:"raw_import_data"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// ProfileRequest is the API request for creating/updating a profile.
type ProfileRequest struct {
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
	PostalCode           *string  `json:"postal_code"`
	PreferredJobTitles   []string `json:"preferred_job_titles"`
	PreferredLocations   []string `json:"preferred_locations"`
	SalaryExpectationMin *int     `json:"salary_expectation_min"`
	SalaryExpectationMax *int     `json:"salary_expectation_max"`
	SalaryCurrency       *string  `json:"salary_currency"`
	WorkAuthorization    *string  `json:"work_authorization"`
	WillingToRelocate    *bool    `json:"willing_to_relocate"`
}

// Experience represents a work experience entry.
type Experience struct {
	ID                 uuid.UUID      `json:"id" db:"id"`
	UserID             uuid.UUID      `json:"user_id" db:"user_id"`
	Title              string         `json:"title" db:"title"`
	CompanyName        string         `json:"company_name" db:"company_name"`
	CompanyLinkedInURL sql.NullString `json:"company_linkedin_url" db:"company_linkedin_url"`
	CompanyLogoURL     sql.NullString `json:"company_logo_url" db:"company_logo_url"`
	EmploymentType     sql.NullString `json:"employment_type" db:"employment_type"`
	Location           sql.NullString `json:"location" db:"location"`
	LocationType       sql.NullString `json:"location_type" db:"location_type"`
	StartDate          time.Time      `json:"start_date" db:"start_date"`
	EndDate            sql.NullTime   `json:"end_date" db:"end_date"`
	IsCurrent          bool           `json:"is_current" db:"is_current"`
	Description        sql.NullString `json:"description" db:"description"`
	Achievements       pq.StringArray `json:"achievements" db:"achievements"`
	SkillsUsed         pq.StringArray `json:"skills_used" db:"skills_used"`
	ImportedFrom       sql.NullString `json:"imported_from" db:"imported_from"`
	DisplayOrder       int            `json:"display_order" db:"display_order"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

// ExperienceRequest is the API request for creating/updating an experience.
type ExperienceRequest struct {
	Title              string   `json:"title" binding:"required"`
	CompanyName        string   `json:"company_name" binding:"required"`
	CompanyLinkedInURL *string  `json:"company_linkedin_url"`
	CompanyLogoURL     *string  `json:"company_logo_url"`
	EmploymentType     *string  `json:"employment_type"`
	Location           *string  `json:"location"`
	LocationType       *string  `json:"location_type"`
	StartDate          string   `json:"start_date" binding:"required"`
	EndDate            *string  `json:"end_date"`
	IsCurrent          bool     `json:"is_current"`
	Description        *string  `json:"description"`
	Achievements       []string `json:"achievements"`
	SkillsUsed         []string `json:"skills_used"`
	DisplayOrder       *int     `json:"display_order"`
}

// Education represents an education entry.
type Education struct {
	ID                 uuid.UUID      `json:"id" db:"id"`
	UserID             uuid.UUID      `json:"user_id" db:"user_id"`
	InstitutionName    string         `json:"institution_name" db:"institution_name"`
	InstitutionLogoURL sql.NullString `json:"institution_logo_url" db:"institution_logo_url"`
	Degree             sql.NullString `json:"degree" db:"degree"`
	FieldOfStudy       sql.NullString `json:"field_of_study" db:"field_of_study"`
	Grade              sql.NullString `json:"grade" db:"grade"`
	StartDate          sql.NullTime   `json:"start_date" db:"start_date"`
	EndDate            sql.NullTime   `json:"end_date" db:"end_date"`
	IsCurrent          bool           `json:"is_current" db:"is_current"`
	Description        sql.NullString `json:"description" db:"description"`
	Activities         pq.StringArray `json:"activities" db:"activities"`
	ImportedFrom       sql.NullString `json:"imported_from" db:"imported_from"`
	DisplayOrder       int            `json:"display_order" db:"display_order"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

// EducationRequest is the API request for creating/updating education.
type EducationRequest struct {
	InstitutionName    string   `json:"institution_name" binding:"required"`
	InstitutionLogoURL *string  `json:"institution_logo_url"`
	Degree             *string  `json:"degree"`
	FieldOfStudy       *string  `json:"field_of_study"`
	Grade              *string  `json:"grade"`
	StartDate          *string  `json:"start_date"`
	EndDate            *string  `json:"end_date"`
	IsCurrent          bool     `json:"is_current"`
	Description        *string  `json:"description"`
	Activities         []string `json:"activities"`
	DisplayOrder       *int     `json:"display_order"`
}

// Skill represents a skill or certification.
type Skill struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	UserID              uuid.UUID      `json:"user_id" db:"user_id"`
	Name                string         `json:"name" db:"name"`
	Category            sql.NullString `json:"category" db:"category"`
	ProficiencyLevel    sql.NullString `json:"proficiency_level" db:"proficiency_level"`
	YearsOfExperience   sql.NullInt32  `json:"years_of_experience" db:"years_of_experience"`
	IsCertification     bool           `json:"is_certification" db:"is_certification"`
	IssuingOrganization sql.NullString `json:"issuing_organization" db:"issuing_organization"`
	IssueDate           sql.NullTime   `json:"issue_date" db:"issue_date"`
	ExpiryDate          sql.NullTime   `json:"expiry_date" db:"expiry_date"`
	CredentialURL       sql.NullString `json:"credential_url" db:"credential_url"`
	ImportedFrom        sql.NullString `json:"imported_from" db:"imported_from"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// SkillRequest is the API request for creating/updating a skill.
type SkillRequest struct {
	Name                string  `json:"name" binding:"required"`
	Category            *string `json:"category"`
	ProficiencyLevel    *string `json:"proficiency_level"`
	YearsOfExperience   *int    `json:"years_of_experience"`
	IsCertification     bool    `json:"is_certification"`
	IssuingOrganization *string `json:"issuing_organization"`
	IssueDate           *string `json:"issue_date"`
	ExpiryDate          *string `json:"expiry_date"`
	CredentialURL       *string `json:"credential_url"`
}

// ResumeData is the complete profile data for CV generation.
type ResumeData struct {
	User        UserResponse `json:"user"`
	Profile     *Profile     `json:"profile"`
	Experiences []Experience `json:"experiences"`
	Education   []Education  `json:"education"`
	Skills      []Skill      `json:"skills"`
}

// AuthResponse is returned after successful authentication.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// RefreshRequest is the request for refreshing tokens.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// CVParseRequest is the request for parsing a CV.
type CVParseRequest struct {
	FileContent string `json:"file_content" binding:"required"`
	FileName    string `json:"file_name" binding:"required"`
	FileType    string `json:"file_type"` // pdf, docx, txt
}

// ParsedCV is the result of CV parsing via Gemini.
type ParsedCV struct {
	FirstName   string             `json:"first_name"`
	LastName    string             `json:"last_name"`
	Email       string             `json:"email"`
	Phone       string             `json:"phone"`
	Location    string             `json:"location"`
	Headline    string             `json:"headline"`
	Summary     string             `json:"summary"`
	LinkedInURL string             `json:"linkedin_url"`
	GithubURL   string             `json:"github_url"`
	Website     string             `json:"website"`
	Experiences []ParsedExperience `json:"experiences"`
	Education   []ParsedEducation  `json:"education"`
	Skills      []ParsedSkill      `json:"skills"`
}

// ParsedExperience is an experience extracted from a CV.
type ParsedExperience struct {
	Title          string   `json:"title"`
	CompanyName    string   `json:"company_name"`
	Location       string   `json:"location"`
	EmploymentType string   `json:"employment_type"`
	StartDate      string   `json:"start_date"`
	EndDate        string   `json:"end_date"`
	IsCurrent      bool     `json:"is_current"`
	Description    string   `json:"description"`
	Achievements   []string `json:"achievements"`
}

// ParsedEducation is education extracted from a CV.
type ParsedEducation struct {
	InstitutionName string   `json:"institution_name"`
	Degree          string   `json:"degree"`
	FieldOfStudy    string   `json:"field_of_study"`
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	Grade           string   `json:"grade"`
	Activities      []string `json:"activities"`
}

// ParsedSkill is a skill extracted from a CV.
type ParsedSkill struct {
	Name             string `json:"name"`
	Category         string `json:"category"`
	ProficiencyLevel string `json:"proficiency_level"`
}
