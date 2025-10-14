package handlers

import (
	"context"
	"net/http"
	"strconv"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type TimeBasedProductHandler struct {
	timeBasedProductService *services.TimeBasedProductService
	enhancedCronService     *services.EnhancedCronService
}

func NewTimeBasedProductHandler(
	timeBasedProductService *services.TimeBasedProductService,
	enhancedCronService *services.EnhancedCronService,
) *TimeBasedProductHandler {
	return &TimeBasedProductHandler{
		timeBasedProductService: timeBasedProductService,
		enhancedCronService:     enhancedCronService,
	}
}

// GetProductsByTimeAdvanced godoc
// @Summary Get products by time with advanced filtering
// @Description Get products available at specific time with filtering options
// @Tags products
// @Accept json
// @Produce json
// @Param restaurant_id query string true "Restaurant ID"
// @Param date_time query string false "DateTime in RFC3339 format (2025-01-02T14:30:00Z)"
// @Param time query string false "Time in HH:MM format (14:30)"
// @Param date query string false "Date in YYYY-MM-DD format (2025-01-02)"
// @Param category query string false "Filter by category"
// @Param tags query array false "Filter by tags"
// @Param price_min query number false "Minimum price filter"
// @Param price_max query number false "Maximum price filter"
// @Param availability query boolean false "Filter by availability"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Success 200 {object} services.TimeBasedProductsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/products/by-time-advanced [get]
func (h *TimeBasedProductHandler) GetProductsByTimeAdvanced(c *gin.Context) {
	// Build request from query parameters
	req := &services.GetProductsByTimeRequest{
		RestaurantID: c.Query("restaurant_id"),
	}

	if req.RestaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant_id parameter",
		})
		return
	}

	// Parse optional parameters
	if dateTime := c.Query("date_time"); dateTime != "" {
		req.DateTime = &dateTime
	}

	if timeParam := c.Query("time"); timeParam != "" {
		req.Time = &timeParam
	}

	if date := c.Query("date"); date != "" {
		req.Date = &date
	}

	if category := c.Query("category"); category != "" {
		req.Category = &category
	}

	// Parse tags array
	if tags := c.QueryArray("tags"); len(tags) > 0 {
		req.Tags = tags
	}

	// Parse price range
	if priceMinStr := c.Query("price_min"); priceMinStr != "" {
		if priceMin, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			if req.PriceRange == nil {
				req.PriceRange = &services.PriceFilter{}
			}
			req.PriceRange.Min = &priceMin
		}
	}

	if priceMaxStr := c.Query("price_max"); priceMaxStr != "" {
		if priceMax, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			if req.PriceRange == nil {
				req.PriceRange = &services.PriceFilter{}
			}
			req.PriceRange.Max = &priceMax
		}
	}

	// Parse availability
	if availabilityStr := c.Query("availability"); availabilityStr != "" {
		if availability, err := strconv.ParseBool(availabilityStr); err == nil {
			req.Availability = &availability
		}
	}

	// Parse pagination
	req.Page = 1
	req.Limit = 20

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	// Call service
	ctx := context.Background()
	response, err := h.timeBasedProductService.GetProductsByTimeAdvanced(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get products",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ForceRestaurantStatusUpdate godoc
// @Summary Force restaurant status update
// @Description Manually trigger status update for a specific restaurant
// @Tags restaurants
// @Accept json
// @Produce json
// @Param id path string true "Restaurant ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/restaurants/{id}/force-status-update [post]
func (h *TimeBasedProductHandler) ForceRestaurantStatusUpdate(c *gin.Context) {
	restaurantID := c.Param("id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	ctx := context.Background()
	if err := h.enhancedCronService.ForceStatusUpdate(ctx, restaurantID); err != nil {
		if err.Error() == "restaurant not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Restaurant not found",
				Message: "The specified restaurant does not exist",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to force status update",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Restaurant status updated successfully",
		"restaurant_id": restaurantID,
	})
}

// GetRestaurantStatusHistory godoc
// @Summary Get restaurant status history
// @Description Get the status change history for a restaurant
// @Tags restaurants
// @Accept json
// @Produce json
// @Param id path string true "Restaurant ID"
// @Param days query int false "Number of days to retrieve (default: 7)"
// @Success 200 {array} services.StatusChange
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/restaurants/{id}/status-history [get]
func (h *TimeBasedProductHandler) GetRestaurantStatusHistory(c *gin.Context) {
	restaurantID := c.Param("id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	// Parse days parameter
	days := 7 // default
	if daysStr := c.Query("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 && parsedDays <= 30 {
			days = parsedDays
		}
	}

	ctx := context.Background()
	history, err := h.enhancedCronService.GetRestaurantStatusHistory(ctx, restaurantID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get status history",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"restaurant_id": restaurantID,
		"days":          days,
		"history":       history,
	})
}

// StartAutomaticStatusManagement godoc
// @Summary Start automatic restaurant status management
// @Description Start the background service for automatic restaurant open/close management
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/system/start-automatic-status [post]
func (h *TimeBasedProductHandler) StartAutomaticStatusManagement(c *gin.Context) {
	if err := h.enhancedCronService.StartAutomaticStatusManagement(); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to start automatic status management",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Automatic restaurant status management started successfully",
		"features": []string{
			"Restaurant status updates: Every minute",
			"Maintenance tasks: Every hour",
			"Daily reports: Every day at midnight",
		},
	})
}

// StopAutomaticStatusManagement godoc
// @Summary Stop automatic restaurant status management
// @Description Stop the background service for automatic restaurant open/close management
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/system/stop-automatic-status [post]
func (h *TimeBasedProductHandler) StopAutomaticStatusManagement(c *gin.Context) {
	h.enhancedCronService.StopAutomaticStatusManagement()

	c.JSON(http.StatusOK, gin.H{
		"message": "Automatic restaurant status management stopped successfully",
	})
}

// GetTimeBasedProductStats godoc
// @Summary Get time-based product statistics
// @Description Get statistics about time-based product availability for a restaurant
// @Tags products
// @Accept json
// @Produce json
// @Param restaurant_id query string true "Restaurant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/products/time-based-stats [get]
func (h *TimeBasedProductHandler) GetTimeBasedProductStats(c *gin.Context) {
	restaurantID := c.Query("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant_id parameter",
		})
		return
	}

	// This would return statistics like:
	// - Total products with time restrictions
	// - Products available by time slot
	// - Most popular time groups
	// - Average product availability duration

	c.JSON(http.StatusOK, gin.H{
		"restaurant_id": restaurantID,
		"stats": gin.H{
			"total_products":             100,           // placeholder
			"time_restricted_products":   45,            // placeholder
			"active_time_groups":         6,             // placeholder
			"most_active_time_slot":      "18:00-22:00", // placeholder
			"average_availability_hours": 8.5,           // placeholder
			"time_groups": []gin.H{
				{
					"name":           "Breakfast",
					"time_range":     "06:00-11:00",
					"product_count":  12,
					"average_orders": 45,
				},
				{
					"name":           "Lunch",
					"time_range":     "11:00-16:00",
					"product_count":  18,
					"average_orders": 78,
				},
				{
					"name":           "Dinner",
					"time_range":     "18:00-23:00",
					"product_count":  25,
					"average_orders": 120,
				},
			},
		},
	})
}

// RegisterRoutes registers all time-based product routes
func (h *TimeBasedProductHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Time-based product routes
	router.GET("/products/by-time-advanced", h.GetProductsByTimeAdvanced)
	router.GET("/products/time-based-stats", h.GetTimeBasedProductStats)

	// Restaurant status management routes
	router.POST("/restaurants/:id/force-status-update", h.ForceRestaurantStatusUpdate)
	router.GET("/restaurants/:id/status-history", h.GetRestaurantStatusHistory)

	// System management routes (admin access recommended)
	router.POST("/system/start-automatic-status", h.StartAutomaticStatusManagement)
	router.POST("/system/stop-automatic-status", h.StopAutomaticStatusManagement)
}
