package handlers

import (
	"context"
	"net/http"
	"strconv"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type RefundHandler struct {
	refundService *services.RefundService
}

func NewRefundHandler(refundService *services.RefundService) *RefundHandler {
	return &RefundHandler{
		refundService: refundService,
	}
}

// RegisterRoutes registers the routes for refund management
func (h *RefundHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	refunds := router.Group("/refunds")

	// Customer routes (create refund requests)
	customerRoutes := refunds.Group("/", authMiddleware.AuthRequired())
	{
		// Create a refund request
		customerRoutes.POST("", h.CreateRefund)
		// Get user's refunds
		customerRoutes.GET("", h.GetRefunds)
		// Get specific refund details
		customerRoutes.GET("/:id", h.GetRefundByID)
	}

	// Admin routes
	adminRoutes := refunds.Group("/", authMiddleware.AuthRequired(), authMiddleware.AdminRequired())
	{
		// Update refund status
		adminRoutes.PUT("/:id/status", h.UpdateRefundStatus)
		// Process a refund
		adminRoutes.POST("/:id/process", h.ProcessRefund)
	}

	// Restaurant routes
	restaurantRoutes := refunds.Group("/restaurant", authMiddleware.AuthRequired(), authMiddleware.RestaurantOwnerRequired())
	{
		// Update refund status for restaurant
		restaurantRoutes.PUT("/:id/status", h.UpdateRefundStatus)
	}
}

// CreateRefund godoc
// @Summary Create a refund request
// @Description Create a new refund request for an order
// @Tags refund
// @Accept json
// @Produce json
// @Param refund body services.CreateRefundRequest true "Refund request data"
// @Success 201 {object} models.Refund
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /refunds [post]
func (h *RefundHandler) CreateRefund(c *gin.Context) {
	var req services.CreateRefundRequest
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
	refund, err := h.refundService.CreateRefund(ctx, userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create refund request",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, refund)
}

// GetRefunds godoc
// @Summary Get refunds
// @Description Get list of refunds with optional filters
// @Tags refund
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status"
// @Success 200 {object} services.RefundListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /refunds [get]
func (h *RefundHandler) GetRefunds(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

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
	response, err := h.refundService.GetRefunds(ctx, userID.(string), page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get refunds",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetRefundByID godoc
// @Summary Get refund by ID
// @Description Get a specific refund by its ID
// @Tags refund
// @Accept json
// @Produce json
// @Param id path string true "Refund ID"
// @Success 200 {object} models.Refund
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /refunds/{id} [get]
func (h *RefundHandler) GetRefundByID(c *gin.Context) {
	refundID := c.Param("id")
	if refundID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Refund ID is required",
			Message: "Please provide a valid refund ID",
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
	refund, err := h.refundService.GetRefundByID(ctx, userID.(string), refundID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Refund not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, refund)
}

// UpdateRefundStatus godoc
// @Summary Update refund status (Admin only)
// @Description Update the status of a refund request
// @Tags refund
// @Accept json
// @Produce json
// @Param id path string true "Refund ID"
// @Param update body services.UpdateRefundStatusRequest true "Status update data"
// @Success 200 {object} models.Refund
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/refunds/{id}/status [put]
func (h *RefundHandler) UpdateRefundStatus(c *gin.Context) {
	refundID := c.Param("id")
	if refundID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Refund ID is required",
			Message: "Please provide a valid refund ID",
		})
		return
	}

	var req services.UpdateRefundStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get admin user ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Admin ID not found",
		})
		return
	}

	ctx := context.Background()
	refund, err := h.refundService.UpdateRefundStatus(ctx, adminID.(string), refundID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update refund status",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, refund)
}

// ProcessRefund godoc
// @Summary Process approved refund (Admin only)
// @Description Process an approved refund and initiate payment gateway refund
// @Tags refund
// @Accept json
// @Produce json
// @Param id path string true "Refund ID"
// @Success 200 {object} models.Refund
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /admin/refunds/{id}/process [post]
func (h *RefundHandler) ProcessRefund(c *gin.Context) {
	refundID := c.Param("id")
	if refundID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Refund ID is required",
			Message: "Please provide a valid refund ID",
		})
		return
	}

	// Get admin user ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Admin ID not found",
		})
		return
	}

	ctx := context.Background()
	refund, err := h.refundService.ProcessRefund(ctx, adminID.(string), refundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process refund",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, refund)
}
