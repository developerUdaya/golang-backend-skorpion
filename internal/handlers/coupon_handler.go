package handlers

import (
	"context"
	"net/http"
	"strconv"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type CouponHandler struct {
	couponService *services.CouponService
}

func NewCouponHandler(couponService *services.CouponService) *CouponHandler {
	return &CouponHandler{
		couponService: couponService,
	}
}

// RegisterRoutes registers the routes for coupon management
func (h *CouponHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	coupons := router.Group("/coupons")

	// Public routes
	coupons.GET("", h.GetCoupons)
	coupons.GET("/:id", h.GetCouponByID)
	coupons.POST("/validate", h.ValidateCoupon)

	// Protected routes (admin/restaurant owner only)
	admin := coupons.Group("/", authMiddleware.AuthRequired(), authMiddleware.AdminRequired())
	{
		admin.POST("", h.CreateCoupon)
		admin.PUT("/:id", h.UpdateCoupon)
		admin.DELETE("/:id", h.DeleteCoupon)
	}
}

// CreateCoupon godoc
// @Summary Create a new coupon
// @Description Create a new coupon for restaurant or platform
// @Tags coupon
// @Accept json
// @Produce json
// @Param coupon body services.CreateCouponRequest true "Coupon data"
// @Success 201 {object} models.Coupon
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /coupons [post]
func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	var req services.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context (admin or restaurant owner)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ctx := context.Background()
	coupon, err := h.couponService.CreateCoupon(ctx, userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create coupon",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, coupon)
}

// GetCoupons godoc
// @Summary Get coupons
// @Description Get list of coupons with optional filters
// @Tags coupon
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param restaurant_id query string false "Filter by restaurant ID"
// @Param active query bool false "Filter by active status"
// @Success 200 {object} services.CouponListResponse
// @Failure 400 {object} ErrorResponse
// @Router /coupons [get]
func (h *CouponHandler) GetCoupons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	restaurantID := c.Query("restaurant_id")
	activeStr := c.Query("active")

	var active *bool
	if activeStr != "" {
		activeBool, err := strconv.ParseBool(activeStr)
		if err == nil {
			active = &activeBool
		}
	}

	ctx := context.Background()
	response, err := h.couponService.GetCoupons(ctx, page, limit, restaurantID, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get coupons",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetCouponByID godoc
// @Summary Get coupon by ID
// @Description Get a specific coupon by its ID
// @Tags coupon
// @Accept json
// @Produce json
// @Param id path string true "Coupon ID"
// @Success 200 {object} models.Coupon
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /coupons/{id} [get]
func (h *CouponHandler) GetCouponByID(c *gin.Context) {
	couponID := c.Param("id")
	if couponID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Coupon ID is required",
			Message: "Please provide a valid coupon ID",
		})
		return
	}

	ctx := context.Background()
	coupon, err := h.couponService.GetCouponByID(ctx, couponID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Coupon not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, coupon)
}

// ValidateCoupon godoc
// @Summary Validate coupon code
// @Description Validate a coupon code for a specific user and restaurant
// @Tags coupon
// @Accept json
// @Produce json
// @Param validation body services.ValidateCouponRequest true "Validation data"
// @Success 200 {object} services.CouponValidationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /coupons/validate [post]
func (h *CouponHandler) ValidateCoupon(c *gin.Context) {
	var req services.ValidateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ctx := context.Background()
	validation, err := h.couponService.ValidateCoupon(ctx, userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Coupon validation failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, validation)
}

// UpdateCoupon godoc
// @Summary Update coupon
// @Description Update an existing coupon
// @Tags coupon
// @Accept json
// @Produce json
// @Param id path string true "Coupon ID"
// @Param coupon body services.UpdateCouponRequest true "Updated coupon data"
// @Success 200 {object} models.Coupon
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /coupons/{id} [put]
func (h *CouponHandler) UpdateCoupon(c *gin.Context) {
	couponID := c.Param("id")
	if couponID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Coupon ID is required",
			Message: "Please provide a valid coupon ID",
		})
		return
	}

	var req services.UpdateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ctx := context.Background()
	coupon, err := h.couponService.UpdateCoupon(ctx, userID.(string), couponID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update coupon",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, coupon)
}

// DeleteCoupon godoc
// @Summary Delete coupon
// @Description Delete a coupon (soft delete - mark as inactive)
// @Tags coupon
// @Accept json
// @Produce json
// @Param id path string true "Coupon ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /coupons/{id} [delete]
func (h *CouponHandler) DeleteCoupon(c *gin.Context) {
	couponID := c.Param("id")
	if couponID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Coupon ID is required",
			Message: "Please provide a valid coupon ID",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ctx := context.Background()
	if err := h.couponService.DeleteCoupon(ctx, userID.(string), couponID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete coupon",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
