package handlers

import (
	"context"
	"net/http"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ShopTimeHandler struct {
	shopTimeService *services.ShopTimeService
}

func NewShopTimeHandler(shopTimeService *services.ShopTimeService) *ShopTimeHandler {
	return &ShopTimeHandler{
		shopTimeService: shopTimeService,
	}
}

// RegisterRoutes registers the routes for shop time management
func (h *ShopTimeHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Public routes
	router.GET("/restaurants/:restaurant_id/shop-timing", h.GetShopTiming)
	router.GET("/time-groups/:group_id/products", h.GetProductsByTime)

	// Protected routes
	restaurant := router.Group("/restaurants/:restaurant_id", authMiddleware.AuthRequired(), authMiddleware.RestaurantRequired())
	{
		// Shop timing routes
		restaurant.PUT("/shop-timing", h.UpdateShopTiming)
		restaurant.PUT("/shop-status", h.UpdateShopStatus)

		// Time group routes
		timeGroups := restaurant.Group("/time-groups")
		{
			timeGroups.POST("", h.CreateTimeGroup)
			timeGroups.GET("", h.GetTimeGroups)
			timeGroups.PUT("/:group_id", h.UpdateTimeGroup)
			timeGroups.DELETE("/:group_id", h.DeleteTimeGroup)
			timeGroups.POST("/:group_id/products", h.AddProductToTimeGroup)
			timeGroups.DELETE("/:group_id/products/:product_id", h.RemoveProductFromTimeGroup)
		}
	}
}

// UpdateShopTiming godoc
// @Summary Update restaurant shop timing
// @Description Update opening hours and auto open/close settings for a restaurant
// @Tags shop-timing
// @Accept json
// @Produce json
// @Param restaurant_id path string true "Restaurant ID"
// @Param timing body services.UpdateShopTimingRequest true "Shop timing data"
// @Success 200 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{restaurant_id}/shop-timing [put]
func (h *ShopTimeHandler) UpdateShopTiming(c *gin.Context) {
	restaurantID := c.Param("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	var req services.UpdateShopTimingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	restaurant, err := h.shopTimeService.UpdateShopTiming(ctx, restaurantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update shop timing",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurant)
}

// GetShopTiming godoc
// @Summary Get restaurant shop timing
// @Description Get opening hours and current status for a restaurant
// @Tags shop-timing
// @Accept json
// @Produce json
// @Param restaurant_id path string true "Restaurant ID"
// @Success 200 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{restaurant_id}/shop-timing [get]
func (h *ShopTimeHandler) GetShopTiming(c *gin.Context) {
	restaurantID := c.Param("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	ctx := context.Background()
	restaurant, err := h.shopTimeService.GetShopTiming(ctx, restaurantID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Restaurant not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurant)
}

// UpdateShopStatus godoc
// @Summary Update restaurant open/close status
// @Description Manually update restaurant open/close status
// @Tags shop-timing
// @Accept json
// @Produce json
// @Param restaurant_id path string true "Restaurant ID"
// @Param status body services.ShopStatusRequest true "Shop status data"
// @Success 200 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{restaurant_id}/status [put]
func (h *ShopTimeHandler) UpdateShopStatus(c *gin.Context) {
	restaurantID := c.Param("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	var req services.ShopStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	restaurant, err := h.shopTimeService.UpdateShopStatus(ctx, restaurantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update shop status",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurant)
}

// CreateTimeGroup godoc
// @Summary Create time-based product group
// @Description Create a time range group for products (e.g., breakfast, lunch, dinner)
// @Tags time-products
// @Accept json
// @Produce json
// @Param group body services.CreateTimeGroupRequest true "Time group data"
// @Success 201 {object} models.TimeRangeProductsGroup
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /time-groups [post]
func (h *ShopTimeHandler) CreateTimeGroup(c *gin.Context) {
	var req services.CreateTimeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	group, err := h.shopTimeService.CreateTimeGroup(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create time group",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// GetTimeGroups godoc
// @Summary Get time-based product groups
// @Description Get all time range groups for a restaurant
// @Tags time-products
// @Accept json
// @Produce json
// @Param restaurant_id query string true "Restaurant ID"
// @Success 200 {array} models.TimeRangeProductsGroup
// @Failure 400 {object} ErrorResponse
// @Router /time-groups [get]
func (h *ShopTimeHandler) GetTimeGroups(c *gin.Context) {
	restaurantID := c.Query("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	ctx := context.Background()
	groups, err := h.shopTimeService.GetTimeGroups(ctx, restaurantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get time groups",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// UpdateTimeGroup godoc
// @Summary Update time-based product group
// @Description Update details of a time range group
// @Tags time-products
// @Accept json
// @Produce json
// @Param group_id path string true "Time Group ID"
// @Param group body services.UpdateTimeGroupRequest true "Updated time group data"
// @Success 200 {object} models.TimeRangeProductsGroup
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /time-groups/{group_id} [put]
func (h *ShopTimeHandler) UpdateTimeGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Group ID is required",
			Message: "Please provide a valid group ID",
		})
		return
	}

	var req services.UpdateTimeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	group, err := h.shopTimeService.UpdateTimeGroup(ctx, groupID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update time group",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteTimeGroup godoc
// @Summary Delete time-based product group
// @Description Delete a time range group
// @Tags time-products
// @Accept json
// @Produce json
// @Param group_id path string true "Time Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /time-groups/{group_id} [delete]
func (h *ShopTimeHandler) DeleteTimeGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Group ID is required",
			Message: "Please provide a valid group ID",
		})
		return
	}

	ctx := context.Background()
	if err := h.shopTimeService.DeleteTimeGroup(ctx, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete time group",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// AddProductToTimeGroup godoc
// @Summary Add product to time group
// @Description Add a product to a time-based product group
// @Tags time-products
// @Accept json
// @Produce json
// @Param group_id path string true "Time Group ID"
// @Param product body services.AddProductToTimeGroupRequest true "Product data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /time-groups/{group_id}/products [post]
func (h *ShopTimeHandler) AddProductToTimeGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Group ID is required",
			Message: "Please provide a valid group ID",
		})
		return
	}

	var req services.AddProductToTimeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := h.shopTimeService.AddProductToTimeGroup(ctx, groupID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to add product to time group",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product added to time group successfully",
	})
}

// RemoveProductFromTimeGroup godoc
// @Summary Remove product from time group
// @Description Remove a product from a time-based product group
// @Tags time-products
// @Accept json
// @Produce json
// @Param group_id path string true "Time Group ID"
// @Param product_id path string true "Product ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /time-groups/{group_id}/products/{product_id} [delete]
func (h *ShopTimeHandler) RemoveProductFromTimeGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	productID := c.Param("product_id")

	if groupID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Group ID and Product ID are required",
			Message: "Please provide valid group ID and product ID",
		})
		return
	}

	ctx := context.Background()
	if err := h.shopTimeService.RemoveProductFromTimeGroup(ctx, groupID, productID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to remove product from time group",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product removed from time group successfully",
	})
}

// GetProductsByTime godoc
// @Summary Get products available at specific time
// @Description Get products that are available at a specific time based on time groups
// @Tags time-products
// @Accept json
// @Produce json
// @Param restaurant_id query string true "Restaurant ID"
// @Param time query string false "Time to check (HH:MM format), defaults to current time"
// @Success 200 {object} services.TimeBasedProductResponse
// @Failure 400 {object} ErrorResponse
// @Router /products/by-time [get]
func (h *ShopTimeHandler) GetProductsByTime(c *gin.Context) {
	restaurantID := c.Query("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide a valid restaurant ID",
		})
		return
	}

	timeParam := c.Query("time")
	var targetTime *string
	if timeParam != "" {
		targetTime = &timeParam
	}

	ctx := context.Background()
	response, err := h.shopTimeService.GetProductsByTime(ctx, restaurantID, targetTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get products by time",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
