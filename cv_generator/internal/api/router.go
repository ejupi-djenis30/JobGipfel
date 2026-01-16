package api

import (
	"github.com/gin-gonic/gin"

	"cv_generator/internal/auth"
	"cv_generator/internal/config"
	"cv_generator/internal/generator"
	"cv_generator/internal/pdf"
)

// SetupRouter configures the Gin router with all routes.
func SetupRouter(
	cfg *config.Config,
	authClient *auth.Client,
	geminiClient *generator.Client,
	pdfConverter *pdf.Converter,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	handler := NewHandler(cfg, authClient, geminiClient, pdfConverter)

	// Health check
	r.GET("/health", handler.HealthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// CV generation (requires auth)
		cv := v1.Group("/cv")
		cv.Use(AuthMiddleware())
		{
			cv.POST("/generate", handler.GenerateCV)
			cv.POST("/preview", handler.PreviewCV)
		}

		// Public endpoints
		v1.GET("/cv/styles", handler.GetStyles)
		v1.GET("/cv/options", handler.GetOptions)
	}

	return r
}
