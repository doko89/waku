package middleware

import (
	"os"
	"strings"
	"waku/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the API token from Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		
		// Check if Authorization header exists
		if authHeader == "" {
			utils.ErrorResponse(c, 401, "Unauthorized: Missing Authorization header")
			c.Abort()
			return
		}
		
		// Check if it starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.ErrorResponse(c, 401, "Unauthorized: Invalid Authorization format. Use 'Bearer <token>'")
			c.Abort()
			return
		}
		
		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		// Get expected token from environment
		expectedToken := os.Getenv("API_TOKEN")
		
		// Validate token
		if token != expectedToken {
			utils.ErrorResponse(c, 401, "Unauthorized: Invalid API token")
			c.Abort()
			return
		}
		
		// Token is valid, continue to next handler
		c.Next()
	}
}

