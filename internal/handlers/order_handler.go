package handlers

import (
	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService *services.OrderService
}

func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// @Summary Create a new order
// @Description Create a new order from cart
// @Tags orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body services.CreateOrderRequest true "Order creation request"
// @Success 201 {object} services.OrderResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req services.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// @Summary Get order by ID
// @Description Get a specific order by its ID
// @Tags orders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderID := c.Param("id")

	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// @Summary Get user orders
// @Description Get all orders for the current user
// @Tags orders
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.Order
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/orders [get]
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, err := h.orderService.GetUserOrders(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// @Summary Get restaurant orders
// @Description Get all orders for a restaurant (restaurant staff/owner only)
// @Tags orders
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.Order
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/restaurant/orders [get]
func (h *OrderHandler) GetRestaurantOrders(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, err := h.orderService.GetRestaurantOrders(c.Request.Context(), restaurantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// @Summary Update order status
// @Description Update the status of an order (restaurant staff/owner only)
// @Tags orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body map[string]string true "Status update request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/restaurant/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	orderID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, req.Status, restaurantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order status updated successfully"})
}

func (h *OrderHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Customer routes
	customer := router.Group("/", authMiddleware.AuthRequired())
	{
		customer.POST("/orders", h.CreateOrder)
		customer.GET("/orders", h.GetUserOrders)
		customer.GET("/orders/:id", h.GetOrderByID)
	}

	// Restaurant routes
	restaurant := router.Group("/restaurant", authMiddleware.AuthRequired(), authMiddleware.RestaurantRequired())
	{
		restaurant.GET("/orders", authMiddleware.RestaurantStaffRequired(), h.GetRestaurantOrders)
		restaurant.PUT("/orders/:id/status", authMiddleware.RestaurantStaffRequired(), h.UpdateOrderStatus)
	}
}
