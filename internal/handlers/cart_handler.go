package handlers

import (
	"context"
	"net/http"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type CartHandler struct {
	cartService CartServiceInterface
}

func NewCartHandler(cartService CartServiceInterface) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

// RegisterRoutes registers the routes for cart management
func (h *CartHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// All cart routes require authentication
	cart := router.Group("/cart", authMiddleware.AuthRequired())
	{
		// Get the user's cart
		cart.GET("", h.GetCart)
		// Add item to cart
		cart.POST("/items", h.AddToCart)
		// Update cart item
		cart.PUT("/items/:item_id", h.UpdateCartItem)
		// Remove item from cart
		cart.DELETE("/items/:item_id", h.RemoveFromCart)
		// Clear cart
		cart.DELETE("", h.ClearCart)
		// Apply coupon
		cart.POST("/coupons", h.ApplyCoupon)
		// Remove coupon
		cart.DELETE("/coupons", h.RemoveCoupon)
		// Get bill summary
		cart.GET("/bill-summary", h.GetBillSummary)
		// Checkout cart
		cart.POST("/checkout", h.Checkout)
	}
}

// GetCart godoc
// @Summary Get user's cart
// @Description Get current user's active cart
// @Tags cart
// @Accept json
// @Produce json
// @Success 200 {object} services.CartResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /cart [get]
func (h *CartHandler) GetCart(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	uid := userID.(string)
	ctx := context.Background()

	// For getting cart, we need to know the restaurant ID
	// Let's assume we get it from query parameter for now
	restaurantID := c.Query("restaurant_id")
	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide restaurant_id parameter",
		})
		return
	}

	cart, err := h.cartService.GetOrCreateCart(ctx, uid, restaurantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get cart",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// AddToCart godoc
// @Summary Add item to cart
// @Description Add or update item in user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Param item body AddToCartRequest true "Cart item data"
// @Success 200 {object} services.CartResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/add [post]
func (h *CartHandler) AddToCart(c *gin.Context) {
	var req AddToCartRequest
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

	uid := userID.(string)
	ctx := context.Background()

	// Create service request
	serviceReq := &services.AddToCartRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	cart, err := h.cartService.AddToCart(ctx, uid, req.RestaurantID, serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to add item to cart",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// UpdateCartItem godoc
// @Summary Update cart item quantity
// @Description Update quantity of an item in cart
// @Tags cart
// @Accept json
// @Produce json
// @Param item body UpdateCartItemRequest true "Update item data"
// @Success 200 {object} services.CartResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/update [put]
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	var req UpdateCartItemRequest
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

	uid := userID.(string)
	ctx := context.Background()

	// Create service request
	serviceReq := &services.UpdateCartItemRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	cart, err := h.cartService.UpdateCartItem(ctx, uid, serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update cart item",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// RemoveFromCart godoc
// @Summary Remove item from cart
// @Description Remove an item from user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Param productId path string true "Product ID"
// @Success 200 {object} services.CartResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/remove/{productId} [delete]
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	productID := c.Param("productId")
	if productID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Product ID is required",
			Message: "Please provide a valid product ID",
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

	uid := userID.(string)
	ctx := context.Background()

	cart, err := h.cartService.RemoveFromCart(ctx, uid, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to remove item from cart",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// ClearCart godoc
// @Summary Clear user's cart
// @Description Remove all items from user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /cart/clear [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	uid := userID.(string)
	ctx := context.Background()

	if err := h.cartService.ClearCart(ctx, uid); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to clear cart",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ApplyCoupon godoc
// @Summary Apply coupon to cart
// @Description Apply a coupon code to user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Param coupon body ApplyCouponRequest true "Coupon data"
// @Success 200 {object} services.CartResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/coupon [post]
func (h *CartHandler) ApplyCoupon(c *gin.Context) {
	var req ApplyCouponRequest
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

	uid := userID.(string)
	ctx := context.Background()

	cart, err := h.cartService.ApplyCoupon(ctx, uid, req.CouponCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to apply coupon",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// RemoveCoupon godoc
// @Summary Remove coupon from cart
// @Description Remove applied coupon from user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Success 200 {object} services.CartResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /cart/coupon [delete]
func (h *CartHandler) RemoveCoupon(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	uid := userID.(string)
	ctx := context.Background()

	cart, err := h.cartService.RemoveCoupon(ctx, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to remove coupon",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// GetBillSummary godoc
// @Summary Get bill summary for cart
// @Description Get detailed bill summary including taxes, delivery charges, and coupons
// @Tags cart
// @Accept json
// @Produce json
// @Param restaurant_id query string true "Restaurant ID"
// @Param address_id query string true "Delivery Address ID"
// @Success 200 {object} services.BillSummaryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/bill-summary [get]
func (h *CartHandler) GetBillSummary(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	uid := userID.(string)
	restaurantID := c.Query("restaurant_id")
	addressID := c.Query("address_id")

	if restaurantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Restaurant ID is required",
			Message: "Please provide restaurant_id parameter",
		})
		return
	}

	if addressID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Address ID is required",
			Message: "Please provide address_id parameter",
		})
		return
	}

	ctx := context.Background()
	billSummary, err := h.cartService.GetBillSummary(ctx, uid, restaurantID, addressID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get bill summary",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, billSummary)
}

// Checkout godoc
// @Summary Checkout cart
// @Description Create order and payment records for cart checkout
// @Tags cart
// @Accept json
// @Produce json
// @Param checkout body CheckoutRequest true "Checkout data"
// @Success 200 {object} services.CheckoutResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /cart/checkout [post]
func (h *CartHandler) Checkout(c *gin.Context) {
	var req CheckoutRequest
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

	uid := userID.(string)
	ctx := context.Background()

	checkoutResponse, err := h.cartService.Checkout(ctx, uid, req.RestaurantID, req.AddressID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to checkout",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, checkoutResponse)
}

// Request and Response structs
type AddToCartRequest struct {
	ProductID    string `json:"product_id" binding:"required"`
	RestaurantID string `json:"restaurant_id" binding:"required"`
	Quantity     int    `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=0"`
}

type ApplyCouponRequest struct {
	CouponCode string `json:"coupon_code" binding:"required"`
}

type CheckoutRequest struct {
	RestaurantID string `json:"restaurant_id" binding:"required"`
	AddressID    string `json:"address_id" binding:"required"`
}

// ErrorResponse is defined in restaurant_handler.go
