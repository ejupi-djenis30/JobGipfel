package models

import (
	"encoding/json"
	"time"
)

// JobSource represents the origin of a job.
type JobSource string

const (
	SourceJobRoom  JobSource = "jobroom"
	SourcePlatform JobSource = "platform"
)

// JobStatus represents the status of a job.
type JobStatus string

const (
	StatusActive   JobStatus = "active"
	StatusInactive JobStatus = "inactive"
	StatusExpired  JobStatus = "expired"
	StatusDraft    JobStatus = "draft"
)

// JobDetail is the root struct for unmarshalling the complete job JSON.
// It mirrors the exact JSON hierarchy from the job-room.ch API.
type JobDetail struct {
	ID                  string      `json:"id" db:"id"`
	CreatedTime         string      `json:"createdTime" db:"created_time"`
	UpdatedTime         string      `json:"updatedTime" db:"updated_time"`
	Status              string      `json:"status" db:"status"`
	SourceSystem        string      `json:"sourceSystem" db:"source_system"`
	ExternalReference   string      `json:"externalReference" db:"external_ref"`
	StellennummerEgov   string      `json:"stellennummerEgov" db:"stellennummer_egov"`
	Fingerprint         *string     `json:"fingerprint,omitempty" db:"fingerprint"`
	ReportingObligation bool        `json:"reportingObligation" db:"reporting_obligation"`
	JobContent          JobContent  `json:"jobContent"`
	Publication         Publication `json:"publication"`
	RawData             string      `json:"-" db:"raw_data"` // Stores the original JSON
}

// ToRawJSON marshals the job to JSON for storage.
func (j *JobDetail) ToRawJSON() ([]byte, error) {
	return json.Marshal(j)
}

// JobContent contains the nested job content data.
type JobContent struct {
	ExternalURL     string           `json:"externalUrl"`
	NumberOfJobs    string           `json:"numberOfJobs"`
	JobDescriptions []JobDescription `json:"jobDescriptions"`
	Company         Company          `json:"company"`
	Employment      Employment       `json:"employment"`
	Location        Location         `json:"location"`
	Occupations     []Occupation     `json:"occupations"`
	ApplyChannel    ApplyChannel     `json:"applyChannel"`
}

// JobDescription contains language-specific title and description.
type JobDescription struct {
	ID              int64  `json:"-" db:"id"`
	JobID           string `json:"-" db:"job_id"`
	LanguageIsoCode string `json:"languageIsoCode" db:"language_iso_code"`
	Title           string `json:"title" db:"title"`
	Description     string `json:"description" db:"description"`
}

// Company contains employer information.
type Company struct {
	ID             int64   `json:"-" db:"id"`
	JobID          string  `json:"-" db:"job_id"`
	Name           string  `json:"name" db:"name"`
	Street         string  `json:"street" db:"street"`
	HouseNumber    *string `json:"houseNumber,omitempty" db:"house_number"`
	PostalCode     string  `json:"postalCode" db:"postal_code"`
	City           string  `json:"city" db:"city"`
	CountryIsoCode string  `json:"countryIsoCode" db:"country_iso_code"`
	Phone          *string `json:"phone,omitempty" db:"phone"`
	Email          *string `json:"email,omitempty" db:"email"`
	Website        *string `json:"website,omitempty" db:"website"`
	Surrogate      bool    `json:"surrogate" db:"surrogate"`
}

// UniqueKey returns a unique key for company deduplication.
func (c *Company) UniqueKey() string {
	return c.Name + "|" + c.PostalCode + "|" + c.City
}

// Employment contains employment terms and conditions.
type Employment struct {
	JobID                 string  `json:"-" db:"job_id"`
	StartDate             *string `json:"startDate" db:"start_date"`
	EndDate               *string `json:"endDate" db:"end_date"`
	ShortEmployment       bool    `json:"shortEmployment" db:"short_employment"`
	Immediately           bool    `json:"immediately" db:"immediately"`
	Permanent             bool    `json:"permanent" db:"permanent"`
	WorkloadPercentageMin string  `json:"workloadPercentageMin" db:"workload_min"`
	WorkloadPercentageMax string  `json:"workloadPercentageMax" db:"workload_max"`
}

// Location contains job location information with embedded coordinates.
type Location struct {
	ID             int64       `json:"-" db:"id"`
	JobID          string      `json:"-" db:"job_id"`
	Remarks        *string     `json:"remarks,omitempty" db:"remarks"`
	City           string      `json:"city" db:"city"`
	PostalCode     string      `json:"postalCode" db:"postal_code"`
	CommunalCode   string      `json:"communalCode" db:"communal_code"`
	RegionCode     string      `json:"regionCode" db:"region_code"`
	CantonCode     string      `json:"cantonCode" db:"canton_code"`
	CountryIsoCode string      `json:"countryIsoCode" db:"country_iso_code"`
	Coordinates    Coordinates `json:"coordinates"`
}

// UniqueKey returns a unique key for location deduplication.
func (l *Location) UniqueKey() string {
	return l.PostalCode + "|" + l.City + "|" + l.CantonCode
}

// Coordinates contains latitude and longitude.
type Coordinates struct {
	Lon string `json:"lon" db:"lon"`
	Lat string `json:"lat" db:"lat"`
}

// LocationDB is a flattened struct for database operations with lat/lon as direct fields.
type LocationDB struct {
	ID             int64   `db:"id"`
	JobID          string  `db:"job_id"`
	Remarks        *string `db:"remarks"`
	City           string  `db:"city"`
	PostalCode     string  `db:"postal_code"`
	CommunalCode   string  `db:"communal_code"`
	RegionCode     string  `db:"region_code"`
	CantonCode     string  `db:"canton_code"`
	CountryIsoCode string  `db:"country_iso_code"`
	Lat            string  `db:"lat"`
	Lon            string  `db:"lon"`
}

// ToLocationDB converts a Location to a flattened LocationDB for database insertion.
func (l *Location) ToLocationDB() LocationDB {
	return LocationDB{
		ID:             l.ID,
		JobID:          l.JobID,
		Remarks:        l.Remarks,
		City:           l.City,
		PostalCode:     l.PostalCode,
		CommunalCode:   l.CommunalCode,
		RegionCode:     l.RegionCode,
		CantonCode:     l.CantonCode,
		CountryIsoCode: l.CountryIsoCode,
		Lat:            l.Coordinates.Lat,
		Lon:            l.Coordinates.Lon,
	}
}

// Occupation contains occupation classification codes.
type Occupation struct {
	ID                 int64   `json:"-" db:"id"`
	JobID              string  `json:"-" db:"job_id"`
	AvamOccupationCode string  `json:"avamOccupationCode" db:"avam_occupation_code"`
	WorkExperience     string  `json:"workExperience" db:"work_experience"`
	EducationCode      *string `json:"educationCode" db:"education_code"`
	QualificationCode  *string `json:"qualificationCode" db:"qualification_code"`
}

// ApplyChannel contains application submission methods.
type ApplyChannel struct {
	JobID          string  `json:"-" db:"job_id"`
	RawPostAddress *string `json:"rawPostAddress,omitempty" db:"raw_post_address"`
	PostAddress    *string `json:"postAddress,omitempty" db:"post_address"`
	EmailAddress   *string `json:"emailAddress" db:"email_address"`
	PhoneNumber    *string `json:"phoneNumber" db:"phone_number"`
	FormURL        *string `json:"formUrl" db:"form_url"`
	AdditionalInfo *string `json:"additionalInfo,omitempty" db:"additional_info"`
}

// Publication contains job publication settings.
type Publication struct {
	JobID             string `json:"-" db:"job_id"`
	StartDate         string `json:"startDate" db:"start_date"`
	EndDate           string `json:"endDate" db:"end_date"`
	EuresDisplay      bool   `json:"euresDisplay" db:"eures_display"`
	PublicDisplay     bool   `json:"publicDisplay" db:"public_display"`
	RestrictedDisplay *bool  `json:"restrictedDisplay,omitempty" db:"restricted_display"`
	CompanyAnonymous  *bool  `json:"companyAnonymous,omitempty" db:"company_anonymous"`
}

// ScrapeRun represents a scraping run for telemetry.
type ScrapeRun struct {
	ID            int64      `json:"id" db:"id"`
	Strategy      string     `json:"strategy" db:"strategy"`
	StartTime     time.Time  `json:"start_time" db:"start_time"`
	EndTime       *time.Time `json:"end_time" db:"end_time"`
	Status        string     `json:"status" db:"status"`
	JobsProcessed int        `json:"jobs_processed" db:"jobs_processed"`
	JobsInserted  int        `json:"jobs_inserted" db:"jobs_inserted"`
	JobsUpdated   int        `json:"jobs_updated" db:"jobs_updated"`
	JobsSkipped   int        `json:"jobs_skipped" db:"jobs_skipped"`
	PagesScraped  int        `json:"pages_scraped" db:"pages_scraped"`
	Filters       *string    `json:"filters" db:"filters"`
	ErrorLog      *string    `json:"error_log" db:"error_log"`
}

// JobSummary is a lightweight representation of a job for listing.
type JobSummary struct {
	ID          string `json:"id" db:"id"`
	Status      string `json:"status" db:"status"`
	Source      string `json:"source" db:"source"`
	CreatedTime string `json:"created_time" db:"created_time"`
	UpdatedTime string `json:"updated_time" db:"updated_time"`
}
