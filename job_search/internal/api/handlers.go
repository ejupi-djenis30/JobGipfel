package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"job_search/internal/config"
	"job_search/internal/models"
	"job_search/internal/search"
)

// Handler holds API handler dependencies.
type Handler struct {
	config        *config.Config
	searchService *search.Service
}

// NewHandler creates a new Handler.
func NewHandler(cfg *config.Config, searchService *search.Service) *Handler {
	return &Handler{
		config:        cfg,
		searchService: searchService,
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "job_search",
		"timestamp": time.Now(),
	})
}

// SearchJobs handles POST /api/v1/jobs/search
func (h *Handler) SearchJobs(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Accept empty body for default search
		req = models.SearchRequest{}
	}

	// Set defaults
	if req.Pagination == nil {
		req.Pagination = &models.PaginationInput{
			Page:    1,
			PerPage: h.config.DefaultPageSize,
		}
	}
	if req.Pagination.PerPage > h.config.MaxPageSize {
		req.Pagination.PerPage = h.config.MaxPageSize
	}

	result, err := h.searchService.Search(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Search failed",
			Code:    "SEARCH_ERROR",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetJob handles GET /api/v1/jobs/:id
func (h *Handler) GetJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Job ID required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	job, err := h.searchService.GetJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get job",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Job not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetFilterOptions handles GET /api/v1/jobs/filters
func (h *Handler) GetFilterOptions(c *gin.Context) {
	options, err := h.searchService.GetFilterOptions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get filter options",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, options)
}

// SaveSearch handles POST /api/v1/saved-searches
func (h *Handler) SaveSearch(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req struct {
		Name   string               `json:"name" binding:"required"`
		Search models.SearchRequest `json:"search"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	saved, err := h.searchService.SaveSearch(c.Request.Context(), userID, req.Name, &req.Search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to save search",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusCreated, saved)
}

// ListSavedSearches handles GET /api/v1/saved-searches
func (h *Handler) ListSavedSearches(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	searches, err := h.searchService.GetSavedSearches(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to list saved searches",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"saved_searches": searches})
}

// DeleteSavedSearch handles DELETE /api/v1/saved-searches/:id
func (h *Handler) DeleteSavedSearch(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid search ID",
			Code:  "INVALID_ID",
		})
		return
	}

	if err := h.searchService.DeleteSavedSearch(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete saved search",
			Code:  "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Saved search deleted"})
}

// GetRecommendations handles POST /api/v1/jobs/recommend
func (h *Handler) GetRecommendations(c *gin.Context) {
	_, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req models.RecommendRequest
	c.ShouldBindJSON(&req)

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}

	// For now, return recent active jobs
	// TODO: Integrate with user profile from auth_service for personalized recommendations
	searchReq := &models.SearchRequest{
		UseAI: true,
		Sort: &models.SortOptions{
			Field: "created_time",
			Order: "desc",
		},
		Pagination: &models.PaginationInput{
			Page:    1,
			PerPage: req.Limit,
		},
	}

	result, err := h.searchService.Search(c.Request.Context(), searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to get recommendations",
			Code:  "SEARCH_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": result.Jobs,
		"total":           result.Total,
	})
}
