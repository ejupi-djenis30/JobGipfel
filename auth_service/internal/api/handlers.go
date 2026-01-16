package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auth_service/internal/auth"
	"auth_service/internal/config"
	"auth_service/internal/gemini"
	"auth_service/internal/models"
	"auth_service/internal/store"
)

// Handler holds API handler dependencies.
type Handler struct {
	config           *config.Config
	store            *store.Store
	jwtManager       *auth.JWTManager
	googleProvider   *auth.GoogleProvider
	linkedInProvider *auth.LinkedInProvider
	geminiClient     *gemini.Client
}

// NewHandler creates a new Handler.
func NewHandler(
	cfg *config.Config,
	store *store.Store,
	jwtManager *auth.JWTManager,
	googleProvider *auth.GoogleProvider,
	linkedInProvider *auth.LinkedInProvider,
	geminiClient *gemini.Client,
) *Handler {
	return &Handler{
		config:           cfg,
		store:            store,
		jwtManager:       jwtManager,
		googleProvider:   googleProvider,
		linkedInProvider: linkedInProvider,
		geminiClient:     geminiClient,
	}
}

// ==================== Health Check ====================

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "auth_service",
		"timestamp": time.Now(),
	})
}

// ==================== OAuth Handlers ====================

// GoogleAuth handles GET /api/v1/auth/google
func (h *Handler) GoogleAuth(c *gin.Context) {
	state := GenerateState()
	// In production, store state in session/cookie for validation
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)

	url := h.googleProvider.GetAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles GET /api/v1/auth/google/callback
func (h *Handler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Authorization code required",
			Code:  "MISSING_CODE",
		})
		return
	}

	// Exchange code for token
	token, err := h.googleProvider.Exchange(c.Request.Context(), code)
	if err != nil {
		slog.Error("Google OAuth exchange failed", "error", err)
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "OAuth exchange failed",
			Code:    "OAUTH_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Get user info
	userInfo, err := h.googleProvider.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		slog.Error("Failed to get Google user info", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get user info",
			Code:    "USER_INFO_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Find or create user
	user, err := h.findOrCreateGoogleUser(c, userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create user",
			Code:    "USER_CREATE_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Generate tokens and respond
	h.respondWithTokens(c, user)
}

// LinkedInAuth handles GET /api/v1/auth/linkedin
func (h *Handler) LinkedInAuth(c *gin.Context) {
	state := GenerateState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)

	url := h.linkedInProvider.GetAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// LinkedInCallback handles GET /api/v1/auth/linkedin/callback
func (h *Handler) LinkedInCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Authorization code required",
			Code:  "MISSING_CODE",
		})
		return
	}

	// Exchange code for token
	token, err := h.linkedInProvider.Exchange(c.Request.Context(), code)
	if err != nil {
		slog.Error("LinkedIn OAuth exchange failed", "error", err)
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "OAuth exchange failed",
			Code:    "OAUTH_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Get user info
	userInfo, err := h.linkedInProvider.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		slog.Error("Failed to get LinkedIn user info", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get user info",
			Code:    "USER_INFO_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Find or create user
	user, err := h.findOrCreateLinkedInUser(c, userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create user",
			Code:    "USER_CREATE_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Update profile with LinkedIn data
	h.updateProfileFromLinkedIn(c, user.ID, userInfo)

	// Generate tokens and respond
	h.respondWithTokens(c, user)
}

// ==================== Token Management ====================

// RefreshToken handles POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Validate refresh token
	tokenHash := auth.HashRefreshToken(req.RefreshToken)
	userID, _, err := h.store.GetRefreshToken(c.Request.Context(), tokenHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid or expired refresh token",
			Code:  "INVALID_TOKEN",
		})
		return
	}

	// Revoke old token
	h.store.RevokeRefreshToken(c.Request.Context(), tokenHash)

	// Get user
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "User not found",
			Code:  "USER_NOT_FOUND",
		})
		return
	}

	// Generate new tokens
	h.respondWithTokens(c, user)
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		tokenHash := auth.HashRefreshToken(req.RefreshToken)
		h.store.RevokeRefreshToken(c.Request.Context(), tokenHash)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ==================== User ====================

// GetMe handles GET /api/v1/me
func (h *Handler) GetMe(c *gin.Context) {
	userID, _ := GetUserID(c)

	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "User not found",
			Code:  "USER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ==================== Profile ====================

// GetProfile handles GET /api/v1/profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID, _ := GetUserID(c)

	profile, err := h.store.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get profile",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	if profile == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Profile not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, _ := GetUserID(c)

	var req models.ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	profile, err := h.store.UpsertProfile(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update profile",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ==================== Experiences ====================

// ListExperiences handles GET /api/v1/experiences
func (h *Handler) ListExperiences(c *gin.Context) {
	userID, _ := GetUserID(c)

	experiences, err := h.store.ListExperiences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to list experiences",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, experiences)
}

// CreateExperience handles POST /api/v1/experiences
func (h *Handler) CreateExperience(c *gin.Context) {
	userID, _ := GetUserID(c)

	var req models.ExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	exp, err := h.store.CreateExperience(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create experience",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, exp)
}

// UpdateExperience handles PUT /api/v1/experiences/:id
func (h *Handler) UpdateExperience(c *gin.Context) {
	userID, _ := GetUserID(c)
	expID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid experience ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Verify ownership
	existing, err := h.store.GetExperience(c.Request.Context(), expID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Experience not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	var req models.ExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	exp, err := h.store.UpdateExperience(c.Request.Context(), expID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update experience",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, exp)
}

// DeleteExperience handles DELETE /api/v1/experiences/:id
func (h *Handler) DeleteExperience(c *gin.Context) {
	userID, _ := GetUserID(c)
	expID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid experience ID",
			Code:  "INVALID_ID",
		})
		return
	}

	// Verify ownership
	existing, err := h.store.GetExperience(c.Request.Context(), expID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Experience not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	if err := h.store.DeleteExperience(c.Request.Context(), expID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete experience",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Experience deleted"})
}

// ==================== Education ====================

// ListEducation handles GET /api/v1/education
func (h *Handler) ListEducation(c *gin.Context) {
	userID, _ := GetUserID(c)

	education, err := h.store.ListEducation(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to list education",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, education)
}

// CreateEducation handles POST /api/v1/education
func (h *Handler) CreateEducation(c *gin.Context) {
	userID, _ := GetUserID(c)

	var req models.EducationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	edu, err := h.store.CreateEducation(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create education",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, edu)
}

// UpdateEducation handles PUT /api/v1/education/:id
func (h *Handler) UpdateEducation(c *gin.Context) {
	userID, _ := GetUserID(c)
	eduID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid education ID",
			Code:  "INVALID_ID",
		})
		return
	}

	existing, err := h.store.GetEducation(c.Request.Context(), eduID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Education not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	var req models.EducationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	edu, err := h.store.UpdateEducation(c.Request.Context(), eduID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update education",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, edu)
}

// DeleteEducation handles DELETE /api/v1/education/:id
func (h *Handler) DeleteEducation(c *gin.Context) {
	userID, _ := GetUserID(c)
	eduID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid education ID",
			Code:  "INVALID_ID",
		})
		return
	}

	existing, err := h.store.GetEducation(c.Request.Context(), eduID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Education not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	if err := h.store.DeleteEducation(c.Request.Context(), eduID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete education",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Education deleted"})
}

// ==================== Skills ====================

// ListSkills handles GET /api/v1/skills
func (h *Handler) ListSkills(c *gin.Context) {
	userID, _ := GetUserID(c)

	skills, err := h.store.ListSkills(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to list skills",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, skills)
}

// CreateSkill handles POST /api/v1/skills
func (h *Handler) CreateSkill(c *gin.Context) {
	userID, _ := GetUserID(c)

	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	skill, err := h.store.CreateSkill(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create skill",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, skill)
}

// UpdateSkill handles PUT /api/v1/skills/:id
func (h *Handler) UpdateSkill(c *gin.Context) {
	userID, _ := GetUserID(c)
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid skill ID",
			Code:  "INVALID_ID",
		})
		return
	}

	existing, err := h.store.GetSkill(c.Request.Context(), skillID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Skill not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	skill, err := h.store.UpdateSkill(c.Request.Context(), skillID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update skill",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, skill)
}

// DeleteSkill handles DELETE /api/v1/skills/:id
func (h *Handler) DeleteSkill(c *gin.Context) {
	userID, _ := GetUserID(c)
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid skill ID",
			Code:  "INVALID_ID",
		})
		return
	}

	existing, err := h.store.GetSkill(c.Request.Context(), skillID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Skill not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	if err := h.store.DeleteSkill(c.Request.Context(), skillID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete skill",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Skill deleted"})
}

// ==================== Import/Export ====================

// ImportCV handles POST /api/v1/import/cv
func (h *Handler) ImportCV(c *gin.Context) {
	userID, _ := GetUserID(c)

	if h.geminiClient == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error: "CV parsing service not configured",
			Code:  "SERVICE_UNAVAILABLE",
		})
		return
	}

	var req models.CVParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Parse CV using Gemini
	parsed, err := h.geminiClient.ParseCV(c.Request.Context(), req.FileContent, req.FileName)
	if err != nil {
		slog.Error("CV parsing failed", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to parse CV",
			Code:    "PARSE_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Import parsed data
	if err := h.importParsedCV(c, userID, parsed); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to import CV data",
			Code:    "IMPORT_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "CV imported successfully",
		"parsed_data": parsed,
	})
}

// ExportResumeData handles GET /api/v1/export/resume-data
func (h *Handler) ExportResumeData(c *gin.Context) {
	userID, _ := GetUserID(c)

	resumeData, err := h.store.GetResumeData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to export resume data",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resumeData)
}

// ==================== Helper Methods ====================

func (h *Handler) findOrCreateGoogleUser(c *gin.Context, info *auth.GoogleUserInfo) (*models.User, error) {
	ctx := c.Request.Context()

	// Try to find by Google ID
	user, err := h.store.GetUserByGoogleID(ctx, info.ID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		h.store.UpdateUserLastLogin(ctx, user.ID)
		return user, nil
	}

	// Try to find by email
	user, err = h.store.GetUserByEmail(ctx, info.Email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Link Google account
		h.store.UpdateUserGoogleID(ctx, user.ID, info.ID)
		h.store.UpdateUserLastLogin(ctx, user.ID)
		return user, nil
	}

	// Create new user
	return h.store.CreateUser(ctx, info.Email, info.VerifiedEmail, &info.ID, nil, &info.Picture)
}

func (h *Handler) findOrCreateLinkedInUser(c *gin.Context, info *auth.LinkedInUserInfo) (*models.User, error) {
	ctx := c.Request.Context()

	// Try to find by LinkedIn ID
	user, err := h.store.GetUserByLinkedInID(ctx, info.ID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		h.store.UpdateUserLastLogin(ctx, user.ID)
		return user, nil
	}

	// Try to find by email
	user, err = h.store.GetUserByEmail(ctx, info.Email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Link LinkedIn account
		h.store.UpdateUserLinkedInID(ctx, user.ID, info.ID)
		h.store.UpdateUserLastLogin(ctx, user.ID)
		return user, nil
	}

	// Create new user
	var picture *string
	if info.Picture != "" {
		picture = &info.Picture
	}
	return h.store.CreateUser(ctx, info.Email, true, nil, &info.ID, picture)
}

func (h *Handler) updateProfileFromLinkedIn(c *gin.Context, userID uuid.UUID, info *auth.LinkedInUserInfo) {
	req := &models.ProfileRequest{
		FirstName: &info.FirstName,
		LastName:  &info.LastName,
	}
	h.store.UpsertProfile(c.Request.Context(), userID, req)
}

func (h *Handler) respondWithTokens(c *gin.Context, user *models.User) {
	tokens, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate tokens",
			Code:  "TOKEN_ERROR",
		})
		return
	}

	// Save refresh token
	tokenHash := auth.HashRefreshToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(h.jwtManager.RefreshExpiry())
	h.store.SaveRefreshToken(c.Request.Context(), user.ID, tokenHash, expiresAt)

	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
		User:         user.ToResponse(),
	})
}

func (h *Handler) importParsedCV(c *gin.Context, userID uuid.UUID, parsed *models.ParsedCV) error {
	ctx := c.Request.Context()

	// Update profile
	req := &models.ProfileRequest{
		FirstName:   nilString(parsed.FirstName),
		LastName:    nilString(parsed.LastName),
		Headline:    nilString(parsed.Headline),
		Summary:     nilString(parsed.Summary),
		Phone:       nilString(parsed.Phone),
		Website:     nilString(parsed.Website),
		LinkedInURL: nilString(parsed.LinkedInURL),
		GithubURL:   nilString(parsed.GithubURL),
	}

	if parsed.Location != "" {
		req.City = &parsed.Location
	}

	h.store.UpsertProfile(ctx, userID, req)

	// Store raw import data
	rawData, _ := json.Marshal(parsed)
	h.store.UpdateProfileImportData(ctx, userID, "cv_parser", rawData)

	// Import experiences
	if len(parsed.Experiences) > 0 {
		h.store.BulkCreateExperiences(ctx, userID, parsed.Experiences, "cv_parser")
	}

	// Import education
	if len(parsed.Education) > 0 {
		h.store.BulkCreateEducation(ctx, userID, parsed.Education, "cv_parser")
	}

	// Import skills
	if len(parsed.Skills) > 0 {
		h.store.BulkCreateSkills(ctx, userID, parsed.Skills, "cv_parser")
	}

	return nil
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ensure sql.NullString is used
var _ = sql.NullString{}
