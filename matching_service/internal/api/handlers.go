package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"matching_service/internal/config"
	"matching_service/internal/matcher"
	"matching_service/internal/models"
)

type Handler struct {
	config         *config.Config
	matcherService *matcher.Service
}

func NewHandler(cfg *config.Config, matcherService *matcher.Service) *Handler {
	return &Handler{
		config:         cfg,
		matcherService: matcherService,
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "matching_service",
		"timestamp": time.Now(),
	})
}

// GetMatches handles GET /api/v1/matches
func (h *Handler) GetMatches(c *gin.Context) {
	accessToken := GetAccessToken(c)

	var req models.MatchRequest
	c.ShouldBindQuery(&req)

	matches, err := h.matcherService.GetMatches(c.Request.Context(), accessToken, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get matches",
			Code:    "MATCHING_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, matches)
}

// GetJobScore handles GET /api/v1/matches/:job_id/score
func (h *Handler) GetJobScore(c *gin.Context) {
	accessToken := GetAccessToken(c)
	jobID := c.Param("job_id")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Job ID required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	score, err := h.matcherService.ScoreJob(c.Request.Context(), accessToken, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to score job",
			Code:    "SCORING_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, score)
}

// AnalyzeProfile handles POST /api/v1/profile/analyze
func (h *Handler) AnalyzeProfile(c *gin.Context) {
	accessToken := GetAccessToken(c)

	analysis, err := h.matcherService.AnalyzeProfile(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to analyze profile",
			Code:    "ANALYSIS_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}
