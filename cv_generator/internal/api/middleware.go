package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"cv_generator/internal/models"
)

// AuthMiddleware validates JWT tokens from the Authorization header.
// Note: This only validates the token format, the actual user data comes from auth_service.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Authorization header required",
				Code:  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid authorization header format",
				Code:  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Parse token without verification (auth_service will validate)
		// We just need to extract the claims to pass along
		parser := jwt.NewParser()
		claims := jwt.MapClaims{}
		_, _, err := parser.ParseUnverified(token, claims)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid token format",
				Code:  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		c.Set("access_token", token)
		c.Next()
	}
}

// GetAccessToken retrieves the access token from context.
func GetAccessToken(c *gin.Context) string {
	token, _ := c.Get("access_token")
	if t, ok := token.(string); ok {
		return t
	}
	return ""
}

// CORSMiddleware handles CORS.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
