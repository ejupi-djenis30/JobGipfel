package api

import (
	"github.com/gin-gonic/gin"

	"job_search/internal/config"
	"job_search/internal/search"
)

// SetupRouter configures the Gin router with all routes.
func SetupRouter(cfg *config.Config, searchService *search.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	handler := NewHandler(cfg, searchService)

	// Health check
	r.GET("/health", handler.HealthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Public search endpoints (no auth required)
		jobs := v1.Group("/jobs")
		jobs.Use(OptionalAuthMiddleware()) // Optional auth for personalization
		{
			jobs.POST("/search", handler.SearchJobs)
			jobs.GET("/:id", handler.GetJob)
			jobs.GET("/filters", handler.GetFilterOptions)
		}

		// Protected endpoints (require auth)
		protected := v1.Group("")
		protected.Use(AuthMiddleware())
		{
			// Saved searches
			protected.POST("/saved-searches", handler.SaveSearch)
			protected.GET("/saved-searches", handler.ListSavedSearches)
			protected.DELETE("/saved-searches/:id", handler.DeleteSavedSearch)

			// Recommendations
			protected.POST("/jobs/recommend", handler.GetRecommendations)
		}
	}

	return r
}
