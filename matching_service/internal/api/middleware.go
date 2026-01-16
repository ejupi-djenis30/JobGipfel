package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"matching_service/internal/models"
)

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

		if userIDStr, ok := claims["user_id"].(string); ok {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
		}

		c.Set("access_token", token)
		c.Next()
	}
}

func GetAccessToken(c *gin.Context) string {
	token, _ := c.Get("access_token")
	if t, ok := token.(string); ok {
		return t
	}
	return ""
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

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
