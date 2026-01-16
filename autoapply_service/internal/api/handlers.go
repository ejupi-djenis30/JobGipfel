package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"autoapply_service/internal/auth"
	"autoapply_service/internal/config"
	"autoapply_service/internal/cvgen"
	"autoapply_service/internal/email"
	"autoapply_service/internal/gemini"
	"autoapply_service/internal/models"
	"autoapply_service/internal/selenium"
	"autoapply_service/internal/store"
)

// Handler holds API handler dependencies.
type Handler struct {
	config       *config.Config
	store        *store.Store
	authClient   *auth.Client
	cvgenClient  *cvgen.Client
	emailSender  *email.Sender
	geminiClient *gemini.Client
	automation   *selenium.Automation
}

// NewHandler creates a new Handler.
func NewHandler(
	cfg *config.Config,
	store *store.Store,
	authClient *auth.Client,
	cvgenClient *cvgen.Client,
	emailSender *email.Sender,
	geminiClient *gemini.Client,
	automation *selenium.Automation,
) *Handler {
	return &Handler{
		config:       cfg,
		store:        store,
		authClient:   authClient,
		cvgenClient:  cvgenClient,
		emailSender:  emailSender,
		geminiClient: geminiClient,
		automation:   automation,
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	authStatus := "ok"
	if err := h.authClient.HealthCheck(c.Request.Context()); err != nil {
		authStatus = "unavailable"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"service":      "autoapply_service",
		"timestamp":    time.Now(),
		"auth_service": authStatus,
		"gemini":       h.geminiClient != nil,
		"smtp":         h.emailSender.IsSMTPConfigured(),
	})
}

// ApplyViaEmail handles POST /api/v1/apply/email
func (h *Handler) ApplyViaEmail(c *gin.Context) {
	userID, _ := GetUserID(c)
	accessToken := GetAccessToken(c)

	// Check rate limit
	count, _ := h.store.CountApplicationsInLastHour(c.Request.Context(), userID)
	if count >= h.config.RateLimitPerHour {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
			Error: "Rate limit exceeded. Please try again later.",
			Code:  "RATE_LIMIT",
		})
		return
	}

	var req models.EmailApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Create application record
	app, err := h.store.CreateApplication(c.Request.Context(), userID, &req, models.MethodEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create application",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	// Update status to in_progress
	h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusInProgress, "")

	// Fetch resume data
	resume, err := h.authClient.GetResumeData(c.Request.Context(), accessToken)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, "Failed to fetch profile data")
		c.JSON(http.StatusBadGateway, models.ErrorResponse{
			Error: "Failed to fetch profile data",
			Code:  "AUTH_SERVICE_ERROR",
		})
		return
	}

	// Generate CV using cv_generator service
	slog.Info("Generating CV", "style", req.CVOptions.Style, "color", req.CVOptions.ColorScheme)
	cvResult, err := h.cvgenClient.GenerateCV(c.Request.Context(), accessToken, req.CVOptions.Style, req.CVOptions.ColorScheme)
	if err != nil {
		slog.Warn("Failed to generate CV, continuing without attachment", "error", err)
	} else {
		// Save CV data to application
		h.store.UpdateApplicationCV(c.Request.Context(), app.ID, cvResult.PDFBase64, cvResult.Style, cvResult.ColorScheme)
		slog.Info("CV generated and stored", "size_bytes", len(cvResult.PDFBytes))
	}

	// Generate cover letter
	slog.Info("Generating cover letter", "job_title", req.JobTitle, "company", req.CompanyName)
	coverLetter, err := h.geminiClient.GenerateCoverLetter(
		c.Request.Context(), resume,
		req.JobTitle, req.CompanyName, req.JobDescription,
		req.CustomMessage, "English",
	)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, "Failed to generate cover letter")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate cover letter",
			Code:    "GEMINI_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Save cover letter
	h.store.UpdateApplicationCoverLetter(c.Request.Context(), app.ID, coverLetter.CoverLetter)

	// Build sender name
	senderName := "Job Applicant"
	if resume.Profile != nil {
		if resume.Profile.FirstName != nil && resume.Profile.LastName != nil {
			senderName = *resume.Profile.FirstName + " " + *resume.Profile.LastName
		}
	}

	// Send email with CV attachment
	emailMsg := &email.EmailMessage{
		To:       req.RecipientEmail,
		Subject:  coverLetter.Subject,
		Body:     coverLetter.CoverLetter,
		FromName: senderName,
	}

	// Attach CV if generated
	if cvResult != nil {
		filename := "Resume.pdf"
		if resume.Profile != nil && resume.Profile.FirstName != nil && resume.Profile.LastName != nil {
			filename = *resume.Profile.FirstName + "_" + *resume.Profile.LastName + "_CV.pdf"
		}
		emailMsg.Attachments = []email.Attachment{
			{
				Filename: filename,
				Content:  cvResult.PDFBytes,
				MimeType: "application/pdf",
			},
		}
	}

	if err := h.emailSender.Send(c.Request.Context(), emailMsg); err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, err.Error())
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to send email",
			Code:    "EMAIL_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Mark as sent
	h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusSent, "")

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "Application sent successfully",
		"application_id": app.ID,
		"cover_letter":   coverLetter.CoverLetter,
		"cv_style":       req.CVOptions.Style,
		"cv_attached":    cvResult != nil,
	})
}

// ApplyViaWeb handles POST /api/v1/apply/web
func (h *Handler) ApplyViaWeb(c *gin.Context) {
	userID, _ := GetUserID(c)
	accessToken := GetAccessToken(c)

	// Check rate limit
	count, _ := h.store.CountApplicationsInLastHour(c.Request.Context(), userID)
	if count >= h.config.RateLimitPerHour {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
			Error: "Rate limit exceeded",
			Code:  "RATE_LIMIT",
		})
		return
	}

	var req models.WebApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Create application record
	app, err := h.store.CreateApplication(c.Request.Context(), userID, &req, models.MethodWeb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create application",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusInProgress, "")

	// Fetch resume data
	resume, err := h.authClient.GetResumeData(c.Request.Context(), accessToken)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, "Failed to fetch profile data")
		c.JSON(http.StatusBadGateway, models.ErrorResponse{
			Error: "Failed to fetch profile data",
			Code:  "AUTH_SERVICE_ERROR",
		})
		return
	}

	// Detect form fields
	slog.Info("Detecting form fields", "url", req.JobURL)
	fields, err := h.automation.DetectFormFields(c.Request.Context(), req.JobURL)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, "Failed to detect form fields")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to analyze job application form",
			Code:    "AUTOMATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Generate form responses
	slog.Info("Generating form responses", "field_count", len(fields))
	responses, err := h.geminiClient.GenerateFormResponses(
		c.Request.Context(), resume,
		req.JobTitle, req.CompanyName, req.JobDescription,
		fields,
	)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, "Failed to generate form responses")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate form responses",
			Code:    "GEMINI_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Apply to job
	slog.Info("Submitting application", "url", req.JobURL)
	result, err := h.automation.ApplyToJob(c.Request.Context(), req.JobURL, responses, nil)
	if err != nil {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, err.Error())
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to submit application",
			Code:    "AUTOMATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	if result.Success {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusSent, "")
	} else {
		h.store.UpdateApplicationStatus(c.Request.Context(), app.ID, models.StatusFailed, result.Message)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        result.Success,
		"message":        result.Message,
		"application_id": app.ID,
	})
}

// ListApplications handles GET /api/v1/applications
func (h *Handler) ListApplications(c *gin.Context) {
	userID, _ := GetUserID(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	apps, err := h.store.ListApplications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to list applications",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	total, _ := h.store.CountApplications(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"applications": apps,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetApplication handles GET /api/v1/applications/:id
func (h *Handler) GetApplication(c *gin.Context) {
	userID, _ := GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid application ID",
			Code:  "INVALID_ID",
		})
		return
	}

	app, err := h.store.GetApplication(c.Request.Context(), appID)
	if err != nil || app == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Application not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Verify ownership
	if app.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Application not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// GenerateCoverLetter handles POST /api/v1/cover-letter/generate
func (h *Handler) GenerateCoverLetter(c *gin.Context) {
	accessToken := GetAccessToken(c)

	var req models.CoverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Fetch resume
	resume, err := h.authClient.GetResumeData(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse{
			Error: "Failed to fetch profile data",
			Code:  "AUTH_SERVICE_ERROR",
		})
		return
	}

	if req.Language == "" {
		req.Language = "English"
	}

	// Generate cover letter
	result, err := h.geminiClient.GenerateCoverLetter(
		c.Request.Context(), resume,
		req.JobTitle, req.CompanyName, req.JobDescription,
		req.CustomMessage, req.Language,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate cover letter",
			Code:    "GEMINI_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
