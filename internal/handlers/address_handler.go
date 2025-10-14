package handlers

import (
	"context"
	"net/http"
	"strconv"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AddressHandler struct {
	addressService *services.AddressService
}

func NewAddressHandler(addressService *services.AddressService) *AddressHandler {
	return &AddressHandler{
		addressService: addressService,
	}
}

// RegisterRoutes registers the routes for address management
func (h *AddressHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	addresses := router.Group("/addresses")

	// Protected routes
	addresses.Use(authMiddleware.AuthRequired())
	{
		// Create a new address
		addresses.POST("", h.CreateAddress)
		// Get all user addresses
		addresses.GET("", h.GetAddresses)
		// Get a specific address
		addresses.GET("/:id", h.GetAddressByID)
		// Update an address
		addresses.PUT("/:id", h.UpdateAddress)
		// Delete an address
		addresses.DELETE("/:id", h.DeleteAddress)
		// Set default address
		addresses.POST("/:id/default", h.SetDefaultAddress)
	}
}

// CreateAddress godoc
// @Summary Create a new address
// @Description Create a new address for the user
// @Tags address
// @Accept json
// @Produce json
// @Param address body services.CreateAddressRequest true "Address data"
// @Success 201 {object} models.Address
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /addresses [post]
func (h *AddressHandler) CreateAddress(c *gin.Context) {
	var req services.CreateAddressRequest
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
	address, err := h.addressService.CreateAddress(ctx, userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create address",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, address)
}

// GetAddresses godoc
// @Summary Get user addresses
// @Description Get all addresses for the current user
// @Tags address
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} services.AddressListResponse
// @Failure 401 {object} ErrorResponse
// @Router /addresses [get]
func (h *AddressHandler) GetAddresses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

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
	response, err := h.addressService.GetAddresses(ctx, userID.(string), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get addresses",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetAddressByID godoc
// @Summary Get address by ID
// @Description Get a specific address by its ID
// @Tags address
// @Accept json
// @Produce json
// @Param id path string true "Address ID"
// @Success 200 {object} models.Address
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /addresses/{id} [get]
func (h *AddressHandler) GetAddressByID(c *gin.Context) {
	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Address ID is required",
			Message: "Please provide a valid address ID",
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
	address, err := h.addressService.GetAddressByID(ctx, userID.(string), addressID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Address not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, address)
}

// UpdateAddress godoc
// @Summary Update address
// @Description Update an existing address
// @Tags address
// @Accept json
// @Produce json
// @Param id path string true "Address ID"
// @Param address body services.UpdateAddressRequest true "Updated address data"
// @Success 200 {object} models.Address
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /addresses/{id} [put]
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Address ID is required",
			Message: "Please provide a valid address ID",
		})
		return
	}

	var req services.UpdateAddressRequest
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
	address, err := h.addressService.UpdateAddress(ctx, userID.(string), addressID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update address",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, address)
}

// DeleteAddress godoc
// @Summary Delete address
// @Description Delete an address
// @Tags address
// @Accept json
// @Produce json
// @Param id path string true "Address ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /addresses/{id} [delete]
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Address ID is required",
			Message: "Please provide a valid address ID",
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
	if err := h.addressService.DeleteAddress(ctx, userID.(string), addressID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete address",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// SetDefaultAddress godoc
// @Summary Set default address
// @Description Set an address as the default address for the user
// @Tags address
// @Accept json
// @Produce json
// @Param id path string true "Address ID"
// @Success 200 {object} models.Address
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /addresses/{id}/default [patch]
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Address ID is required",
			Message: "Please provide a valid address ID",
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
	address, err := h.addressService.SetDefaultAddress(ctx, userID.(string), addressID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to set default address",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, address)
}
