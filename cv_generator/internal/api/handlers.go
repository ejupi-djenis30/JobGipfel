package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"cv_generator/internal/auth"
	"cv_generator/internal/config"
	"cv_generator/internal/generator"
	"cv_generator/internal/models"
	"cv_generator/internal/pdf"
)

// Handler holds API handler dependencies.
type Handler struct {
	config       *config.Config
	authClient   *auth.Client
	geminiClient *generator.Client
	pdfConverter *pdf.Converter
}

// NewHandler creates a new Handler.
func NewHandler(
	cfg *config.Config,
	authClient *auth.Client,
	geminiClient *generator.Client,
	pdfConverter *pdf.Converter,
) *Handler {
	return &Handler{
		config:       cfg,
		authClient:   authClient,
		geminiClient: geminiClient,
		pdfConverter: pdfConverter,
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	// Check auth service connectivity
	authStatus := "ok"
	if err := h.authClient.HealthCheck(c.Request.Context()); err != nil {
		authStatus = "unavailable"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"service":      "cv_generator",
		"timestamp":    time.Now(),
		"auth_service": authStatus,
		"gemini":       h.geminiClient != nil,
	})
}

// GetStyles handles GET /api/v1/cv/styles
func (h *Handler) GetStyles(c *gin.Context) {
	styles := []gin.H{
		{
			"id":          "modern",
			"name":        "Modern",
			"description": "Clean, contemporary design with subtle accents and sans-serif fonts",
		},
		{
			"id":          "minimalist",
			"name":        "Minimalist",
			"description": "Ultra-clean with lots of whitespace and minimal decorations",
		},
		{
			"id":          "classic",
			"name":        "Classic",
			"description": "Traditional professional resume with serif fonts and formal layout",
		},
		{
			"id":          "creative",
			"name":        "Creative",
			"description": "Bold colors, unique layouts, and creative typography",
		},
	}
	c.JSON(http.StatusOK, gin.H{"styles": styles})
}

// GetOptions handles GET /api/v1/cv/options
func (h *Handler) GetOptions(c *gin.Context) {
	c.JSON(http.StatusOK, models.GetAvailableOptions())
}

// GenerateCV handles POST /api/v1/cv/generate
func (h *Handler) GenerateCV(c *gin.Context) {
	if h.geminiClient == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error: "CV generation service not configured",
			Code:  "SERVICE_UNAVAILABLE",
		})
		return
	}

	// Parse request
	var req models.GenerateCVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = models.DefaultCVRequest()
	}

	// Apply defaults for missing values
	applyDefaults(&req)

	// Get access token
	accessToken := GetAccessToken(c)

	// Fetch resume data from auth_service
	slog.Info("Fetching resume data from auth_service")
	resumeData, err := h.authClient.GetResumeData(c.Request.Context(), accessToken)
	if err != nil {
		slog.Error("Failed to fetch resume data", "error", err)
		c.JSON(http.StatusBadGateway, models.ErrorResponse{
			Error:   "Failed to fetch profile data",
			Code:    "AUTH_SERVICE_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Validate resume data has content
	if resumeData.Profile == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Profile is empty. Please complete your profile first.",
			Code:  "EMPTY_PROFILE",
		})
		return
	}

	// Generate HTML using Gemini
	slog.Info("Generating CV with Gemini", "style", req.Style)
	html, err := h.geminiClient.GenerateCV(c.Request.Context(), resumeData, &req)
	if err != nil {
		slog.Error("Failed to generate CV HTML", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate CV",
			Code:    "GENERATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Convert HTML to PDF
	slog.Info("Converting HTML to PDF")
	pdfBytes, err := h.pdfConverter.ConvertHTMLToPDF(c.Request.Context(), html)
	if err != nil {
		slog.Error("Failed to convert to PDF", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate PDF",
			Code:    "PDF_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Generate filename
	filename := "resume.pdf"
	if resumeData.Profile.FirstName != nil && resumeData.Profile.LastName != nil {
		filename = fmt.Sprintf("%s_%s_CV.pdf", *resumeData.Profile.FirstName, *resumeData.Profile.LastName)
	}

	// Return PDF
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// PreviewCV handles POST /api/v1/cv/preview
func (h *Handler) PreviewCV(c *gin.Context) {
	if h.geminiClient == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error: "CV generation service not configured",
			Code:  "SERVICE_UNAVAILABLE",
		})
		return
	}

	// Parse request
	var req models.GenerateCVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = models.DefaultCVRequest()
	}

	applyDefaults(&req)

	// Get access token
	accessToken := GetAccessToken(c)

	// Fetch resume data
	resumeData, err := h.authClient.GetResumeData(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse{
			Error:   "Failed to fetch profile data",
			Code:    "AUTH_SERVICE_ERROR",
			Details: err.Error(),
		})
		return
	}

	if resumeData.Profile == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Profile is empty. Please complete your profile first.",
			Code:  "EMPTY_PROFILE",
		})
		return
	}

	// Generate HTML only (no PDF)
	html, err := h.geminiClient.GenerateCV(c.Request.Context(), resumeData, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to generate CV",
			Code:    "GENERATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Return HTML for preview
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// applyDefaults fills in default values for missing request fields.
func applyDefaults(req *models.GenerateCVRequest) {
	if req.Style == "" {
		req.Style = models.StyleModern
	}
	if req.ColorScheme == "" {
		req.ColorScheme = models.ColorBlue
	}
	if req.Language == "" {
		req.Language = "en"
	}
	if req.MaxExperiences <= 0 {
		req.MaxExperiences = 5
	}
	if req.MaxEducation <= 0 {
		req.MaxEducation = 3
	}
	if req.MaxSkills <= 0 {
		req.MaxSkills = 15
	}
	// Default sections to true if all are false
	if !req.Sections.Summary && !req.Sections.Experiences && !req.Sections.Education && !req.Sections.Skills {
		req.Sections = models.CVSections{
			Summary:        true,
			Experiences:    true,
			Education:      true,
			Skills:         true,
			Certifications: true,
		}
	}
}
