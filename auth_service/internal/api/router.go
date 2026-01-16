package api

import (
	"github.com/gin-gonic/gin"

	"auth_service/internal/auth"
	"auth_service/internal/config"
	"auth_service/internal/gemini"
	"auth_service/internal/store"
)

// SetupRouter configures the Gin router with all routes.
func SetupRouter(
	cfg *config.Config,
	store *store.Store,
	jwtManager *auth.JWTManager,
	googleProvider *auth.GoogleProvider,
	linkedInProvider *auth.LinkedInProvider,
	geminiClient *gemini.Client,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware(cfg.FrontendURL))

	handler := NewHandler(cfg, store, jwtManager, googleProvider, linkedInProvider, geminiClient)

	// Health check
	r.GET("/health", handler.HealthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		authGroup := v1.Group("/auth")
		{
			// Google OAuth
			if googleProvider != nil {
				authGroup.GET("/google", handler.GoogleAuth)
				authGroup.GET("/google/callback", handler.GoogleCallback)
			}

			// LinkedIn OAuth
			if linkedInProvider != nil {
				authGroup.GET("/linkedin", handler.LinkedInAuth)
				authGroup.GET("/linkedin/callback", handler.LinkedInCallback)
			}

			// Token management
			authGroup.POST("/refresh", handler.RefreshToken)
			authGroup.POST("/logout", handler.Logout)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(AuthMiddleware(jwtManager))
		{
			// User
			protected.GET("/me", handler.GetMe)

			// Profile
			protected.GET("/profile", handler.GetProfile)
			protected.PUT("/profile", handler.UpdateProfile)

			// Experiences
			protected.GET("/experiences", handler.ListExperiences)
			protected.POST("/experiences", handler.CreateExperience)
			protected.PUT("/experiences/:id", handler.UpdateExperience)
			protected.DELETE("/experiences/:id", handler.DeleteExperience)

			// Education
			protected.GET("/education", handler.ListEducation)
			protected.POST("/education", handler.CreateEducation)
			protected.PUT("/education/:id", handler.UpdateEducation)
			protected.DELETE("/education/:id", handler.DeleteEducation)

			// Skills
			protected.GET("/skills", handler.ListSkills)
			protected.POST("/skills", handler.CreateSkill)
			protected.PUT("/skills/:id", handler.UpdateSkill)
			protected.DELETE("/skills/:id", handler.DeleteSkill)

			// Import/Export
			protected.POST("/import/cv", handler.ImportCV)
			protected.GET("/export/resume-data", handler.ExportResumeData)
		}
	}

	return r
}
