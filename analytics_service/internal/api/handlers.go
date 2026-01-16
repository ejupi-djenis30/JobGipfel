package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"analytics_service/internal/analytics"
	"analytics_service/internal/config"
	"analytics_service/internal/models"
)

type Handler struct {
	config           *config.Config
	analyticsService *analytics.Service
}

func NewHandler(cfg *config.Config, analyticsService *analytics.Service) *Handler {
	return &Handler{
		config:           cfg,
		analyticsService: analyticsService,
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "analytics_service",
		"timestamp": time.Now(),
	})
}

// GetDashboard handles GET /api/v1/dashboard
func (h *Handler) GetDashboard(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "User ID not found",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	dashboard, err := h.analyticsService.GetDashboard(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get dashboard",
			Code:    "ANALYTICS_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// GetApplicationStats handles GET /api/v1/stats/applications
func (h *Handler) GetApplicationStats(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "User ID not found",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	stats, err := h.analyticsService.GetApplicationStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get application stats",
			Code:  "ANALYTICS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetMarketStats handles GET /api/v1/stats/market
func (h *Handler) GetMarketStats(c *gin.Context) {
	stats, err := h.analyticsService.GetMarketStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get market stats",
			Code:  "ANALYTICS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSkillsTrend handles GET /api/v1/stats/skills
func (h *Handler) GetSkillsTrend(c *gin.Context) {
	skills, err := h.analyticsService.GetSkillsTrend(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get skills trend",
			Code:  "ANALYTICS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

// GetActivity handles GET /api/v1/activity
func (h *Handler) GetActivity(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "User ID not found",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	activity, err := h.analyticsService.GetUserActivity(c.Request.Context(), userID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get activity",
			Code:  "ANALYTICS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activity": activity})
}
