package handlers

import (
	"context"
	"io"
	"net/http"

	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type RazorpayHandler struct {
	razorpayService *services.RazorpayService
}

func NewRazorpayHandler(razorpayService *services.RazorpayService) *RazorpayHandler {
	return &RazorpayHandler{
		razorpayService: razorpayService,
	}
}

// PlaceOrder godoc
// @Summary Place a new order with Razorpay integration
// @Description Create order and Razorpay payment order, returns razorpay_order_id for frontend payment
// @Tags orders
// @Accept json
// @Produce json
// @Param order body services.PlaceOrderRequest true "Order placement request"
// @Success 201 {object} services.PlaceOrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/orders/place [post]
func (h *RazorpayHandler) PlaceOrder(c *gin.Context) {
	var req services.PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	response, err := h.razorpayService.CreateRazorpayOrder(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create order",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// PaymentWebhook godoc
// @Summary Handle Razorpay payment webhooks
// @Description Process payment success/failure webhooks from Razorpay and update order status
// @Tags payments
// @Accept json
// @Produce json
// @Param X-Razorpay-Signature header string true "Razorpay webhook signature"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments/webhook [post]
func (h *RazorpayHandler) PaymentWebhook(c *gin.Context) {
	signature := c.GetHeader("X-Razorpay-Signature")
	if signature == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing webhook signature",
			Message: "X-Razorpay-Signature header is required",
		})
		return
	}

	// Read the raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to read request body",
			Message: err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := h.razorpayService.HandlePaymentWebhook(ctx, body, signature); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process webhook",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Webhook processed successfully",
	})
}

// RegisterRoutes registers all Razorpay-related routes
func (h *RazorpayHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Order placement endpoint (without auth for now, as it's called from frontend)
	router.POST("/orders/place", h.PlaceOrder)

	// Webhook endpoint (no auth needed for webhooks from Razorpay)
	router.POST("/webhooks/razorpay", h.PaymentWebhook)
}
