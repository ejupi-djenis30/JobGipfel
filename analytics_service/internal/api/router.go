package api

import (
	"github.com/gin-gonic/gin"

	"analytics_service/internal/analytics"
	"analytics_service/internal/config"
)

func SetupRouter(cfg *config.Config, analyticsService *analytics.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	handler := NewHandler(cfg, analyticsService)

	r.GET("/health", handler.HealthCheck)

	v1 := r.Group("/api/v1")
	v1.Use(AuthMiddleware())
	{
		v1.GET("/dashboard", handler.GetDashboard)
		v1.GET("/stats/applications", handler.GetApplicationStats)
		v1.GET("/stats/market", handler.GetMarketStats)
		v1.GET("/stats/skills", handler.GetSkillsTrend)
		v1.GET("/activity", handler.GetActivity)
	}

	return r
}
