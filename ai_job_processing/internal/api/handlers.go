package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ai_job_processing/internal/config"
	"ai_job_processing/internal/models"
	"ai_job_processing/internal/processor"
)

// Handler holds API handler dependencies.
type Handler struct {
	processor *processor.Processor
	config    *config.Config
	version   string
}

// NewHandler creates a new Handler.
func NewHandler(proc *processor.Processor, cfg *config.Config, version string) *Handler {
	return &Handler{
		processor: proc,
		config:    cfg,
		version:   version,
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	pendingCount, _ := h.processor.GetPendingJobsCount(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"timestamp":    time.Now(),
		"version":      h.version,
		"pending_jobs": pendingCount,
	})
}

// GetLanguages handles GET /api/v1/languages
func (h *Handler) GetLanguages(c *gin.Context) {
	languages := h.processor.GetTargetLanguages()

	languageInfos := make([]models.LanguageInfo, len(languages))
	for i, code := range languages {
		languageInfos[i] = models.LanguageInfo{
			Code: code,
			Name: config.GetLanguageName(code),
		}
	}

	c.JSON(http.StatusOK, models.LanguagesResponse{
		Languages:       languageInfos,
		DefaultLanguage: h.config.SourceLanguage,
	})
}

// Process handles POST /api/v1/process
// Normalizes and translates raw job data. Does not use database.
func (h *Handler) Process(c *gin.Context) {
	var req models.ProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	resp, err := h.processor.Process(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Processing failed",
			Code:    "PROCESSING_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ProcessByID handles POST /api/v1/process/:id
// Normalizes and translates a job from database, saves back.
// Skips if already normalized and all languages translated (unless force=true).
func (h *Handler) ProcessByID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Job ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	var req models.ProcessByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = models.ProcessByIDRequest{}
	}
	req.JobID = jobID

	resp, err := h.processor.ProcessByID(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		code := "PROCESSING_ERROR"

		if err.Error() == "job not found: "+jobID ||
			err.Error() == "failed to load job: job not found: "+jobID {
			statusCode = http.StatusNotFound
			code = "JOB_NOT_FOUND"
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:   "Processing failed",
			Code:    code,
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// TranslateByID handles POST /api/v1/translate/:id
// Translates a job's raw description WITHOUT normalizing.
// Skips languages that already have translations (unless force=true).
func (h *Handler) TranslateByID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Job ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	var req models.TranslateByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = models.TranslateByIDRequest{}
	}
	req.JobID = jobID

	resp, err := h.processor.TranslateByID(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		code := "TRANSLATION_ERROR"

		if err.Error() == "job not found: "+jobID ||
			err.Error() == "failed to load job: job not found: "+jobID {
			statusCode = http.StatusNotFound
			code = "JOB_NOT_FOUND"
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:   "Translation failed",
			Code:    code,
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// NormalizeByID handles POST /api/v1/normalize/:id
// Normalizes a job from database WITHOUT translating.
// Skips if already normalized (unless force=true).
func (h *Handler) NormalizeByID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Job ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	var req models.NormalizeByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = models.NormalizeByIDRequest{}
	}
	req.JobID = jobID

	resp, err := h.processor.NormalizeByID(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		code := "NORMALIZATION_ERROR"

		if err.Error() == "job not found: "+jobID ||
			err.Error() == "failed to load job: job not found: "+jobID {
			statusCode = http.StatusNotFound
			code = "JOB_NOT_FOUND"
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:   "Normalization failed",
			Code:    code,
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Normalize handles POST /api/v1/normalize
// Normalizes raw job data without translation. Does not use database.
func (h *Handler) Normalize(c *gin.Context) {
	var req models.NormalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	resp, err := h.processor.Normalize(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Normalization failed",
			Code:    "NORMALIZATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Translate handles POST /api/v1/translate
// Translates raw job data (normalized or raw description). Does not use database.
func (h *Handler) Translate(c *gin.Context) {
	var req models.TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	resp, err := h.processor.Translate(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Translation failed",
			Code:    "TRANSLATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPendingJobs handles GET /api/v1/pending
func (h *Handler) GetPendingJobs(c *gin.Context) {
	limit := 50

	jobs, err := h.processor.GetPendingJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get pending jobs",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	count, _ := h.processor.GetPendingJobsCount(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
		"total": count,
	})
}
