package api

import (
	"github.com/gin-gonic/gin"

	"autoapply_service/internal/auth"
	"autoapply_service/internal/config"
	"autoapply_service/internal/cvgen"
	"autoapply_service/internal/email"
	"autoapply_service/internal/gemini"
	"autoapply_service/internal/selenium"
	"autoapply_service/internal/store"
)

// SetupRouter configures the Gin router with all routes.
func SetupRouter(
	cfg *config.Config,
	store *store.Store,
	authClient *auth.Client,
	cvgenClient *cvgen.Client,
	emailSender *email.Sender,
	geminiClient *gemini.Client,
	automation *selenium.Automation,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	handler := NewHandler(cfg, store, authClient, cvgenClient, emailSender, geminiClient, automation)

	// Health check
	r.GET("/health", handler.HealthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Protected routes
		protected := v1.Group("")
		protected.Use(AuthMiddleware())
		{
			// Apply endpoints
			protected.POST("/apply/email", handler.ApplyViaEmail)
			protected.POST("/apply/web", handler.ApplyViaWeb)

			// Applications management
			protected.GET("/applications", handler.ListApplications)
			protected.GET("/applications/:id", handler.GetApplication)

			// Cover letter generation
			protected.POST("/cover-letter/generate", handler.GenerateCoverLetter)
		}
	}

	return r
}
