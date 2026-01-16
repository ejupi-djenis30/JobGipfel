package api

import (
	"github.com/gin-gonic/gin"

	"matching_service/internal/config"
	"matching_service/internal/matcher"
)

func SetupRouter(cfg *config.Config, matcherService *matcher.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	handler := NewHandler(cfg, matcherService)

	r.GET("/health", handler.HealthCheck)

	v1 := r.Group("/api/v1")
	v1.Use(AuthMiddleware())
	{
		v1.GET("/matches", handler.GetMatches)
		v1.GET("/matches/:job_id/score", handler.GetJobScore)
		v1.POST("/profile/analyze", handler.AnalyzeProfile)
	}

	return r
}
