package api

import (
	"github.com/gin-gonic/gin"

	"ai_job_processing/internal/config"
	"ai_job_processing/internal/processor"
)

// NewRouter creates and configures the Gin router.
func NewRouter(proc *processor.Processor, cfg *config.Config, version string) *gin.Engine {
	if cfg.LogLevel == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(Recovery())
	router.Use(RequestLogger())
	router.Use(CORS())

	handler := NewHandler(proc, cfg, version)

	// Health check
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Languages
		v1.GET("/languages", handler.GetLanguages)

		// Pending jobs
		v1.GET("/pending", handler.GetPendingJobs)

		// Raw data processing (no database)
		v1.POST("/process", handler.Process)     // Normalize + translate
		v1.POST("/normalize", handler.Normalize) // Normalize only
		v1.POST("/translate", handler.Translate) // Translate only

		// Database processing by job ID
		v1.POST("/process/:id", handler.ProcessByID)     // Normalize + translate, save to DB
		v1.POST("/normalize/:id", handler.NormalizeByID) // Normalize only, save to DB
		v1.POST("/translate/:id", handler.TranslateByID) // Translate only, save to DB
	}

	return router
}
