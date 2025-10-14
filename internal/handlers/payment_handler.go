package handlers

import (
	"net/http"
	"strconv"

	"golang-food-backend/internal/models"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePayment godoc
// @Summary Create a new payment
// @Description Create a new payment for an order
// @Tags payments
// @Accept json
// @Produce json
// @Param payment body CreatePaymentRequest true "Payment data"
// @Success 201 {object} models.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
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

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid order ID",
			Message: err.Error(),
		})
		return
	}

	payment := &models.Payment{
		OrderID:       orderID,
		UserID:        uid,
		Amount:        req.Amount,
		Method:        req.Method,
		TransactionID: req.TransactionID,
	}

	if err := h.paymentService.CreatePayment(payment); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create payment",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

// GetPaymentByID godoc
// @Summary Get payment by ID
// @Description Get payment details by ID
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} models.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /payments/{id} [get]
func (h *PaymentHandler) GetPaymentByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid payment ID",
			Message: err.Error(),
		})
		return
	}

	payment, err := h.paymentService.GetPaymentByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Payment not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// GetPaymentByOrderID godoc
// @Summary Get payment by order ID
// @Description Get payment details by order ID
// @Tags payments
// @Accept json
// @Produce json
// @Param orderId path string true "Order ID"
// @Success 200 {object} models.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /payments/order/{orderId} [get]
func (h *PaymentHandler) GetPaymentByOrderID(c *gin.Context) {
	orderIDStr := c.Param("orderId")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid order ID",
			Message: err.Error(),
		})
		return
	}

	payment, err := h.paymentService.GetPaymentByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Payment not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// UpdatePaymentStatus godoc
// @Summary Update payment status
// @Description Update payment status (success, failed, etc.)
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param status body UpdatePaymentStatusRequest true "Status data"
// @Success 200 {object} models.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /payments/{id}/status [put]
func (h *PaymentHandler) UpdatePaymentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid payment ID",
			Message: err.Error(),
		})
		return
	}

	var req UpdatePaymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	payment, err := h.paymentService.UpdatePaymentStatus(id, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update payment status",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// GetPaymentsByStatus godoc
// @Summary Get payments by status
// @Description Get payments filtered by status with pagination
// @Tags payments
// @Accept json
// @Produce json
// @Param status query string true "Payment status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} PaymentsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments [get]
func (h *PaymentHandler) GetPaymentsByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Status parameter is required",
			Message: "Please provide a status parameter",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	payments, total, err := h.paymentService.GetPaymentsByStatus(status, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch payments",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaymentsResponse{
		Payments: payments,
		Pagination: PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

// ProcessWebhook godoc
// @Summary Process payment webhook
// @Description Process payment gateway webhook
// @Tags payments
// @Accept json
// @Produce json
// @Param webhook body PaymentWebhookRequest true "Webhook data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/webhook [post]
func (h *PaymentHandler) ProcessWebhook(c *gin.Context) {
	var req PaymentWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid webhook data",
			Message: err.Error(),
		})
		return
	}

	if err := h.paymentService.ProcessWebhook(&req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process webhook",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Webhook processed successfully",
	})
}

// Request/Response models
type CreatePaymentRequest struct {
	OrderID       string  `json:"order_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
	Method        string  `json:"method" binding:"required"`
	TransactionID string  `json:"transaction_id" binding:"required"`
}

type UpdatePaymentStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type PaymentsResponse struct {
	Payments   []models.Payment   `json:"payments"`
	Pagination PaginationResponse `json:"pagination"`
}

type PaymentWebhookRequest struct {
	PaymentID     string                 `json:"payment_id"`
	TransactionID string                 `json:"transaction_id"`
	Status        string                 `json:"status"`
	Amount        float64                `json:"amount"`
	Gateway       string                 `json:"gateway"`
	Metadata      map[string]interface{} `json:"metadata"`
}
