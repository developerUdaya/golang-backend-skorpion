package middleware

import (
	"net/http"
	"strings"

	"golang-food-backend/pkg/auth"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// AuthRequired middleware validates JWT token
func (a *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := a.jwtManager.ValidateToken(tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("restaurant_id", claims.RestaurantID)
		c.Set("role", claims.Role)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// RestaurantRequired middleware ensures the user belongs to a restaurant
func (a *AuthMiddleware) RestaurantRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		restaurantID, exists := c.Get("restaurant_id")
		if !exists || restaurantID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RoleRequired middleware checks if user has required role
func (a *AuthMiddleware) RoleRequired(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role information missing"})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// AdminRequired middleware ensures user is an admin
func (a *AuthMiddleware) AdminRequired() gin.HandlerFunc {
	return a.RoleRequired("admin")
}

// RestaurantOwnerRequired middleware ensures user is a restaurant owner
func (a *AuthMiddleware) RestaurantOwnerRequired() gin.HandlerFunc {
	return a.RoleRequired("restaurant_owner", "admin")
}

// RestaurantStaffRequired middleware allows restaurant staff and owners
func (a *AuthMiddleware) RestaurantStaffRequired() gin.HandlerFunc {
	return a.RoleRequired("restaurant_staff", "restaurant_owner", "admin")
}

// GetUserID helper function to extract user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return ""
}

// GetRestaurantID helper function to extract restaurant ID from context
func GetRestaurantID(c *gin.Context) string {
	if restaurantID, exists := c.Get("restaurant_id"); exists {
		return restaurantID.(string)
	}
	return ""
}

// GetUserRole helper function to extract user role from context
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}
