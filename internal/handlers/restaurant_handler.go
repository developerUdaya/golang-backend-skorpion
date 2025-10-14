package handlers

import (
	"net/http"
	"strconv"

	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RestaurantHandler struct {
	restaurantService *services.RestaurantService
}

func NewRestaurantHandler(restaurantService *services.RestaurantService) *RestaurantHandler {
	return &RestaurantHandler{
		restaurantService: restaurantService,
	}
}

// CreateRestaurant godoc
// @Summary Create a new restaurant
// @Description Create a new restaurant with owner details
// @Tags restaurants
// @Accept json
// @Produce json
// @Param restaurant body CreateRestaurantRequest true "Restaurant data"
// @Success 201 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /restaurants [post]
func (h *RestaurantHandler) CreateRestaurant(c *gin.Context) {
	var req CreateRestaurantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ownerID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	restaurant := &models.Restaurant{
		Name:          req.Name,
		Description:   req.Description,
		Logo:          req.Logo,
		CuisineTypes:  req.CuisineTypes,
		OwnerID:       ownerID,
		GSTNumber:     req.GSTNumber,
		ContactNumber: req.ContactNumber,
	}

	if err := h.restaurantService.CreateRestaurant(restaurant); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create restaurant",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, restaurant)
}

// GetRestaurants godoc
// @Summary Get all restaurants
// @Description Get all restaurants with pagination and filtering
// @Tags restaurants
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param cuisine query string false "Filter by cuisine type"
// @Param search query string false "Search by name or description"
// @Success 200 {object} RestaurantsResponse
// @Failure 500 {object} ErrorResponse
// @Router /restaurants [get]
func (h *RestaurantHandler) GetRestaurants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	cuisine := c.Query("cuisine")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	restaurants, total, err := h.restaurantService.GetRestaurants(page, limit, cuisine, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch restaurants",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, RestaurantsResponse{
		Restaurants: restaurants,
		Pagination: PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

// GetRestaurantByID godoc
// @Summary Get restaurant by ID
// @Description Get restaurant details by ID
// @Tags restaurants
// @Accept json
// @Produce json
// @Param id path string true "Restaurant ID"
// @Success 200 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{id} [get]
func (h *RestaurantHandler) GetRestaurantByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid restaurant ID",
			Message: err.Error(),
		})
		return
	}

	restaurant, err := h.restaurantService.GetRestaurantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Restaurant not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurant)
}

// UpdateRestaurant godoc
// @Summary Update restaurant
// @Description Update restaurant details
// @Tags restaurants
// @Accept json
// @Produce json
// @Param id path string true "Restaurant ID"
// @Param restaurant body UpdateRestaurantRequest true "Restaurant data"
// @Success 200 {object} models.Restaurant
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{id} [put]
func (h *RestaurantHandler) UpdateRestaurant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid restaurant ID",
			Message: err.Error(),
		})
		return
	}

	var req UpdateRestaurantRequest
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

	ownerID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	// Check if user owns the restaurant
	restaurant, err := h.restaurantService.GetRestaurantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Restaurant not found",
			Message: err.Error(),
		})
		return
	}

	if restaurant.OwnerID != ownerID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to update this restaurant",
		})
		return
	}

	// Update restaurant
	if req.Name != "" {
		restaurant.Name = req.Name
	}
	if req.Description != "" {
		restaurant.Description = req.Description
	}
	if req.Logo != "" {
		restaurant.Logo = req.Logo
	}
	if len(req.CuisineTypes) > 0 {
		restaurant.CuisineTypes = req.CuisineTypes
	}
	if req.ContactNumber != "" {
		restaurant.ContactNumber = req.ContactNumber
	}
	if req.Status != "" {
		restaurant.Status = req.Status
	}

	if err := h.restaurantService.UpdateRestaurant(restaurant); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update restaurant",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurant)
}

// DeleteRestaurant godoc
// @Summary Delete restaurant
// @Description Soft delete restaurant (set status to inactive)
// @Tags restaurants
// @Accept json
// @Produce json
// @Param id path string true "Restaurant ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /restaurants/{id} [delete]
func (h *RestaurantHandler) DeleteRestaurant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid restaurant ID",
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

	ownerID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	// Check if user owns the restaurant
	restaurant, err := h.restaurantService.GetRestaurantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Restaurant not found",
			Message: err.Error(),
		})
		return
	}

	if restaurant.OwnerID != ownerID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "You don't have permission to delete this restaurant",
		})
		return
	}

	if err := h.restaurantService.DeleteRestaurant(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete restaurant",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMyRestaurants godoc
// @Summary Get current user's restaurants
// @Description Get restaurants owned by the current user
// @Tags restaurants
// @Accept json
// @Produce json
// @Success 200 {array} models.Restaurant
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /restaurants/my [get]
func (h *RestaurantHandler) GetMyRestaurants(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found",
		})
		return
	}

	ownerID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	restaurants, err := h.restaurantService.GetRestaurantsByOwner(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch restaurants",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, restaurants)
}

// RegisterRoutes registers all restaurant routes
func (h *RestaurantHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Public routes
	router.GET("/restaurants", h.GetRestaurants)
	router.GET("/restaurants/:id", h.GetRestaurantByID)

	// Protected routes
	protected := router.Group("/", authMiddleware.AuthRequired())
	{
		protected.POST("/restaurants", h.CreateRestaurant)
		protected.PUT("/restaurants/:id", h.UpdateRestaurant)
		protected.DELETE("/restaurants/:id", h.DeleteRestaurant)
		protected.GET("/restaurants/my", h.GetMyRestaurants)
	}
}

// Request/Response models
type CreateRestaurantRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	Logo          string   `json:"logo"`
	CuisineTypes  []string `json:"cuisine_types" binding:"required"`
	GSTNumber     string   `json:"gst_number" binding:"required"`
	ContactNumber string   `json:"contact_number" binding:"required"`
}

type UpdateRestaurantRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Logo          string   `json:"logo"`
	CuisineTypes  []string `json:"cuisine_types"`
	ContactNumber string   `json:"contact_number"`
	Status        string   `json:"status"`
}

type RestaurantsResponse struct {
	Restaurants []models.Restaurant `json:"restaurants"`
	Pagination  PaginationResponse  `json:"pagination"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
