package handlers

import (
	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productService  ProductServiceInterface
	categoryService CategoryServiceInterface
}

func NewProductHandler(productService ProductServiceInterface, categoryService CategoryServiceInterface) *ProductHandler {
	return &ProductHandler{
		productService:  productService,
		categoryService: categoryService,
	}
}

// @Summary Create a new product
// @Description Create a new product for a restaurant
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body services.CreateProductRequest true "Product creation request"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	var req services.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.productService.CreateProduct(c.Request.Context(), restaurantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// @Summary Get products by restaurant
// @Description Get all products for a specific restaurant
// @Tags products
// @Produce json
// @Param id path string true "Restaurant ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Success 200 {object} PaginatedProductsResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/restaurants/{id}/products [get]
func (h *ProductHandler) GetProductsByRestaurant(c *gin.Context) {
	restaurantID := c.Param("id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	products, err := h.productService.GetProductsByRestaurant(c.Request.Context(), restaurantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// @Summary Search products
// @Description Search products within a restaurant
// @Tags products
// @Produce json
// @Param id path string true "Restaurant ID"
// @Param query query string true "Search query"
// @Param category query string false "Category filter"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Success 200 {object} PaginatedProductsResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/restaurants/{id}/products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	restaurantID := c.Param("id")
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	products, err := h.productService.SearchProducts(c.Request.Context(), restaurantID, query, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// @Summary Get product by ID
// @Description Get a specific product by its ID
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/products/{id} [get]
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	productID := c.Param("id")

	product, err := h.productService.GetProductByID(c.Request.Context(), productID)
	if err != nil {
		// Check for not found error
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":       "Product not found",
				"product_id":  productID,
				"description": "No product exists with the provided ID.",
			})
			return
		}
		// Other errors
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Failed to retrieve product",
			"details":    err.Error(),
			"product_id": productID,
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

// @Summary Get products by restaurant, category and time
// @Description Get paginated products filtered by restaurant, category, availability and current time
// @Tags products
// @Produce json
// @Param restaurant_id path string true "Restaurant ID"
// @Param category_id query string false "Category ID filter"
// @Param available_only query bool false "Show only available products (default: false)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} services.GetProductsResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/restaurants/{restaurant_id}/products/filtered [get]
func (h *ProductHandler) GetProductsByRestaurantCategoryAndTime(c *gin.Context) {
	restaurantID := c.Param("id")

	// Parse query parameters
	categoryID := c.Query("category_id")
	availableOnly, _ := strconv.ParseBool(c.DefaultQuery("available_only", "false"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Build request
	req := &services.GetProductsRequest{
		RestaurantID:  restaurantID,
		CategoryID:    categoryID,
		AvailableOnly: availableOnly,
		Page:          page,
		Limit:         limit,
	}

	// Get products from service
	response, err := h.productService.GetProductsByRestaurantCategoryAndTime(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Failed to retrieve products",
			"details":       err.Error(),
			"restaurant_id": restaurantID,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Update product
// @Description Update an existing product
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body map[string]interface{} true "Product updates"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	productID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.productService.UpdateProduct(c.Request.Context(), productID, restaurantID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// @Summary Delete product
// @Description Delete an existing product
// @Tags products
// @Security BearerAuth
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	productID := c.Param("id")

	if err := h.productService.DeleteProduct(c.Request.Context(), productID, restaurantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// Category handlers

// @Summary Create a new category
// @Description Create a new product category for a restaurant
// @Tags categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body services.CreateCategoryRequest true "Category creation request"
// @Success 201 {object} models.ProductCategory
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/categories [post]
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	var req services.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.categoryService.CreateCategory(c.Request.Context(), restaurantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// @Summary Get categories by restaurant
// @Description Get all categories for a specific restaurant
// @Tags categories
// @Produce json
// @Param id path string true "Restaurant ID"
// @Success 200 {array} string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/restaurants/{id}/categories [get]
func (h *ProductHandler) GetCategoriesByRestaurant(c *gin.Context) {
	restaurantID := c.Param("id")

	categories, err := h.categoryService.GetCategoriesByRestaurant(c.Request.Context(), restaurantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// @Summary Update category
// @Description Update an existing category
// @Tags categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param request body map[string]interface{} true "Category updates"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/categories/{id} [put]
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	categoryID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.categoryService.UpdateCategory(c.Request.Context(), categoryID, restaurantID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
}

// @Summary Delete category
// @Description Delete a category (only if it has no products)
// @Tags categories
// @Security BearerAuth
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/categories/{id} [delete]
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	categoryID := c.Param("id")

	if err := h.categoryService.DeleteCategory(c.Request.Context(), categoryID, restaurantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// @Summary Delete category with products
// @Description Delete a category and all its products
// @Tags categories
// @Security BearerAuth
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/categories/{id}/with-products [delete]
func (h *ProductHandler) DeleteCategoryWithProducts(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	categoryID := c.Param("id")

	if err := h.categoryService.DeleteCategoryWithProducts(c.Request.Context(), categoryID, restaurantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category and its products deleted successfully"})
}

// @Summary Move products and delete category
// @Description Move all products from one category to another and delete the source category
// @Tags categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Source Category ID"
// @Param target_id query string true "Target Category ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/categories/{id}/move-products [delete]
func (h *ProductHandler) MoveProductsAndDeleteCategory(c *gin.Context) {
	restaurantID := middleware.GetRestaurantID(c)
	if restaurantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Restaurant access required"})
		return
	}

	categoryID := c.Param("id")
	targetCategoryID := c.Query("target_id")

	if targetCategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target category ID is required"})
		return
	}

	if err := h.categoryService.MoveProductsAndDeleteCategory(c.Request.Context(), categoryID, targetCategoryID, restaurantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Products moved and category deleted successfully"})
}

func (h *ProductHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// Public routes
	router.GET("/restaurants/:id/products", h.GetProductsByRestaurant)
	router.GET("/restaurants/:id/products/filtered", h.GetProductsByRestaurantCategoryAndTime)
	router.GET("/restaurants/:id/products/search", h.SearchProducts)
	router.GET("/restaurants/:id/categories", h.GetCategoriesByRestaurant)
	router.GET("/products/:id", h.GetProductByID)

	// Protected routes (restaurant staff/owner only)
	protected := router.Group("/", authMiddleware.AuthRequired(), authMiddleware.RestaurantRequired())
	{
		// Product management
		protected.POST("/products", authMiddleware.RestaurantStaffRequired(), h.CreateProduct)
		protected.PUT("/products/:id", authMiddleware.RestaurantStaffRequired(), h.UpdateProduct)
		protected.DELETE("/products/:id", authMiddleware.RestaurantOwnerRequired(), h.DeleteProduct)

		// Category management
		protected.POST("/categories", authMiddleware.RestaurantStaffRequired(), h.CreateCategory)
		protected.PUT("/categories/:id", authMiddleware.RestaurantStaffRequired(), h.UpdateCategory)
		protected.DELETE("/categories/:id", authMiddleware.RestaurantOwnerRequired(), h.DeleteCategory)
		protected.DELETE("/categories/:id/with-products", authMiddleware.RestaurantOwnerRequired(), h.DeleteCategoryWithProducts)
		protected.DELETE("/categories/:id/move-products", authMiddleware.RestaurantOwnerRequired(), h.MoveProductsAndDeleteCategory)
	}
}
