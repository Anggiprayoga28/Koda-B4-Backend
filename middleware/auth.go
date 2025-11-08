package middleware

import (
	"coffee-shop/models"
	"coffee-shop/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Message: "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Message: "Invalid or expired token",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Success: false,
				Message: "User role not found",
			})
			c.Abort()
			return
		}

		if role != "admin" {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Success: false,
				Message: "Access denied. Admin role required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
