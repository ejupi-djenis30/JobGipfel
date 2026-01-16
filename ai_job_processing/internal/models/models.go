package models

import "time"

// ProcessRequest is the request to process raw job data (no database).
type ProcessRequest struct {
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
	SourceLanguage string `json:"source_language,omitempty"`
}

// ProcessByIDRequest is the request to process a job by its database ID.
type ProcessByIDRequest struct {
	JobID           string   `json:"job_id" binding:"required"`
	SourceLanguage  string   `json:"source_language,omitempty"`
	TargetLanguages []string `json:"target_languages,omitempty"`
	Force           bool     `json:"force,omitempty"`
}

// TranslateByIDRequest is the request to translate a job by ID without normalizing.
type TranslateByIDRequest struct {
	JobID           string   `json:"job_id" binding:"required"`
	SourceLanguage  string   `json:"source_language,omitempty"`
	TargetLanguages []string `json:"target_languages,omitempty"`
	Force           bool     `json:"force,omitempty"`
}

// NormalizeByIDRequest is the request to normalize a job by ID without translating.
type NormalizeByIDRequest struct {
	JobID          string `json:"job_id" binding:"required"`
	SourceLanguage string `json:"source_language,omitempty"`
	Force          bool   `json:"force,omitempty"`
}

// ProcessResponse is the response after processing a job.
type ProcessResponse struct {
	JobID        string              `json:"job_id,omitempty"`
	Normalized   *NormalizedContent  `json:"normalized,omitempty"`
	Translations []TranslatedContent `json:"translations,omitempty"`
	ProcessedAt  time.Time           `json:"processed_at"`
	SavedToDB    bool                `json:"saved_to_db,omitempty"`
	Skipped      bool                `json:"skipped,omitempty"`
	SkipReason   string              `json:"skip_reason,omitempty"`
}

// NormalizedContent contains the structured job description sections.
type NormalizedContent struct {
	Tasks        string `json:"tasks"`
	Requirements string `json:"requirements"`
	Offer        string `json:"offer"`
}

// TranslatedContent contains translated content for one language.
type TranslatedContent struct {
	Language     string `json:"language"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Tasks        string `json:"tasks,omitempty"`
	Requirements string `json:"requirements,omitempty"`
	Offer        string `json:"offer,omitempty"`
}

// NormalizeRequest is the request to normalize raw job data (no translation).
type NormalizeRequest struct {
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
	SourceLanguage string `json:"source_language,omitempty"`
}

// NormalizeResponse is the response after normalization.
type NormalizeResponse struct {
	JobID          string             `json:"job_id,omitempty"`
	SourceLanguage string             `json:"source_language"`
	Normalized     *NormalizedContent `json:"normalized"`
	ProcessedAt    time.Time          `json:"processed_at"`
	SavedToDB      bool               `json:"saved_to_db,omitempty"`
	Skipped        bool               `json:"skipped,omitempty"`
	SkipReason     string             `json:"skip_reason,omitempty"`
}

// TranslateRequest is the request to translate content (raw data).
type TranslateRequest struct {
	Title           string             `json:"title" binding:"required"`
	Normalized      *NormalizedContent `json:"normalized,omitempty"`
	Description     string             `json:"description,omitempty"`
	SourceLanguage  string             `json:"source_language,omitempty"`
	TargetLanguages []string           `json:"target_languages,omitempty"`
}

// TranslateResponse is the response after translation.
type TranslateResponse struct {
	JobID            string              `json:"job_id,omitempty"`
	Translations     []TranslatedContent `json:"translations"`
	ProcessedAt      time.Time           `json:"processed_at"`
	SavedToDB        bool                `json:"saved_to_db,omitempty"`
	Skipped          bool                `json:"skipped,omitempty"`
	SkipReason       string              `json:"skip_reason,omitempty"`
	SkippedLanguages []string            `json:"skipped_languages,omitempty"`
}

// LanguagesResponse lists available target languages.
type LanguagesResponse struct {
	Languages       []LanguageInfo `json:"languages"`
	DefaultLanguage string         `json:"default_language,omitempty"`
}

// LanguageInfo describes a supported language.
type LanguageInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// JobFromDB represents a job loaded from the database.
type JobFromDB struct {
	ID           string `db:"id"`
	Title        string `db:"title"`
	Description  string `db:"description"`
	Language     string `db:"language_iso_code"`
	IsNormalized bool   `db:"is_normalized"`
}

// BuildDescription combines normalized sections into a formatted description.
func (n *NormalizedContent) BuildDescription(lang string) string {
	headers := map[string]struct{ tasks, requirements, offer string }{
		"de": {"Aufgaben", "Anforderungen", "Wir bieten"},
		"fr": {"Tâches", "Exigences", "Nous offrons"},
		"it": {"Mansioni", "Requisiti", "Offriamo"},
		"en": {"Tasks", "Requirements", "We Offer"},
	}

	h, ok := headers[lang]
	if !ok {
		h = headers["en"]
	}

	desc := ""
	if n.Tasks != "" {
		desc += "## " + h.tasks + "\n\n" + n.Tasks + "\n\n"
	}
	if n.Requirements != "" {
		desc += "## " + h.requirements + "\n\n" + n.Requirements + "\n\n"
	}
	if n.Offer != "" {
		desc += "## " + h.offer + "\n\n" + n.Offer + "\n\n"
	}

	return desc
}
