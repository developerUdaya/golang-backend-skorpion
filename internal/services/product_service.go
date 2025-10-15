package services

import (
	"context"
	"errors"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/cache"
	"golang-food-backend/pkg/messaging"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductService struct {
	productRepo   repositories.ProductRepository
	categoryRepo  repositories.ProductCategoryRepository
	inventoryRepo repositories.InventoryRepository
	cache         *cache.RedisCache
	kafkaProducer *messaging.KafkaProducer
	kafkaBrokers  []string
}

func NewProductService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.ProductCategoryRepository,
	inventoryRepo repositories.InventoryRepository,
	cache *cache.RedisCache,
	kafkaProducer *messaging.KafkaProducer,
	kafkaBrokers []string,
) *ProductService {
	return &ProductService{
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		inventoryRepo: inventoryRepo,
		cache:         cache,
		kafkaProducer: kafkaProducer,
		kafkaBrokers:  kafkaBrokers,
	}
}

type CreateProductRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	CategoryID      string                 `json:"category_id" binding:"required"`
	Price           float64                `json:"price" binding:"required,gt=0"`
	DiscountPrice   *float64               `json:"discount_price,omitempty"`
	ImageUrls       []string               `json:"image_urls"`
	PreparationTime int                    `json:"preparation_time"`
	Tags            []string               `json:"tags"`
	VideoUrl        string                 `json:"video_url,omitempty"`
	NutritionalInfo map[string]interface{} `json:"nutritional_info,omitempty"`
	InitialStock    int                    `json:"initial_stock"`
	MinStockLevel   int                    `json:"min_stock_level"`
}

func (s *ProductService) CreateProduct(ctx context.Context, restaurantID string, req *CreateProductRequest) (*models.Product, error) {
	// Parse category ID
	categoryObjectID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, errors.New("invalid category ID")
	}

	// Verify category belongs to restaurant
	category, err := s.categoryRepo.GetByID(ctx, categoryObjectID)
	if err != nil {
		return nil, errors.New("category not found")
	}
	if category.RestaurantID != restaurantID {
		return nil, errors.New("category does not belong to this restaurant")
	}

	// Create product
	product := &models.Product{
		RestaurantID:    restaurantID,
		CategoryID:      categoryObjectID,
		Name:            req.Name,
		Description:     req.Description,
		Price:           req.Price,
		DiscountPrice:   req.DiscountPrice,
		ImageUrls:       req.ImageUrls,
		IsAvailable:     true,
		PreparationTime: req.PreparationTime,
		Tags:            req.Tags,
		VideoUrl:        req.VideoUrl,
		NutritionalInfo: req.NutritionalInfo,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	// Create inventory record
	inventory := &models.Inventory{
		ProductID:        product.ID,
		RestaurantID:     restaurantID,
		Quantity:         req.InitialStock,
		ReservedQuantity: 0,
		MinStockLevel:    req.MinStockLevel,
		MaxStockLevel:    req.InitialStock * 2, // Default max stock
		LastRestocked:    time.Now(),
	}

	if err := s.inventoryRepo.Create(ctx, inventory); err != nil {
		return nil, err
	}

	// Send kafka event for product creation
	event := messaging.InventoryEvent{
		Type:         "product_created",
		ProductID:    product.ID.Hex(),
		Quantity:     req.InitialStock,
		RestaurantID: restaurantID,
	}
	s.kafkaProducer.SendMessage("inventory_events", s.kafkaBrokers, product.ID.Hex(), event)

	// Clear cache
	s.clearProductCache(restaurantID)

	return product, nil
}

func (s *ProductService) GetProductsByRestaurant(ctx context.Context, restaurantID string, limit, offset int) ([]models.Product, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("products:%s:%d:%d", restaurantID, limit, offset)
	var cachedProducts []models.Product
	if err := s.cache.Get(ctx, cacheKey, &cachedProducts); err == nil {
		return cachedProducts, nil
	}

	// Get from database
	products, err := s.productRepo.GetByRestaurantID(ctx, restaurantID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache for 15 minutes
	s.cache.Set(ctx, cacheKey, products, time.Minute*15)

	return products, nil
}

func (s *ProductService) SearchProducts(ctx context.Context, restaurantID, query string, limit, offset int) ([]models.Product, error) {
	return s.productRepo.Search(ctx, query, restaurantID, limit, offset)
}

func (s *ProductService) GetProductByID(ctx context.Context, productID string) (*models.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, errors.New("invalid product ID")
	}

	// Try cache first
	cacheKey := "product:" + productID
	var cachedProduct models.Product
	if err := s.cache.Get(ctx, cacheKey, &cachedProduct); err == nil {
		return &cachedProduct, nil
	}

	// Get from database
	product, err := s.productRepo.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}

	// Cache for 30 minutes
	s.cache.Set(ctx, cacheKey, product, time.Minute*30)

	return product, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, productID string, restaurantID string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return errors.New("invalid product ID")
	}

	product, err := s.productRepo.GetByID(ctx, objectID)
	if err != nil {
		return err
	}

	if product.RestaurantID != restaurantID {
		return errors.New("product does not belong to this restaurant")
	}

	// Update allowed fields
	if name, ok := updates["name"]; ok {
		if nameStr, ok := name.(string); ok {
			product.Name = nameStr
		}
	}
	if description, ok := updates["description"]; ok {
		if descStr, ok := description.(string); ok {
			product.Description = descStr
		}
	}
	if price, ok := updates["price"]; ok {
		if priceFloat, ok := price.(float64); ok {
			product.Price = priceFloat
		}
	}
	if isAvailable, ok := updates["is_available"]; ok {
		if availBool, ok := isAvailable.(bool); ok {
			product.IsAvailable = availBool
		}
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return err
	}

	// Clear caches
	s.cache.Delete(ctx, "product:"+productID)
	s.clearProductCache(restaurantID)

	return nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, productID string, restaurantID string) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return errors.New("invalid product ID")
	}

	product, err := s.productRepo.GetByID(ctx, objectID)
	if err != nil {
		return err
	}

	if product.RestaurantID != restaurantID {
		return errors.New("product does not belong to this restaurant")
	}

	if err := s.productRepo.Delete(ctx, objectID); err != nil {
		return err
	}

	// Clear caches
	s.cache.Delete(ctx, "product:"+productID)
	s.clearProductCache(restaurantID)

	return nil
}

type GetProductsRequest struct {
	RestaurantID  string `json:"restaurant_id" binding:"required"`
	CategoryID    string `json:"category_id,omitempty"`
	AvailableOnly bool   `json:"available_only"`
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
}

type GetProductsResponse struct {
	Products   []models.Product `json:"products"`
	Pagination PaginationInfo   `json:"pagination"`
	TimeInfo   TimeAvailability `json:"time_info"`
}

type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type TimeAvailability struct {
	CurrentTime   string `json:"current_time"`
	CurrentDate   string `json:"current_date"`
	IsBusinessDay bool   `json:"is_business_day"`
}

func (s *ProductService) GetProductsByRestaurantCategoryAndTime(ctx context.Context, req *GetProductsRequest) (*GetProductsResponse, error) {
	// Set default pagination values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Maximum limit
	}

	offset := (req.Page - 1) * req.Limit

	// Get current time info
	now := time.Now()
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")

	// Parse category ID if provided
	var categoryID *primitive.ObjectID
	if req.CategoryID != "" {
		objID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category ID")
		}
		categoryID = &objID
	}

	// Try cache first
	cacheKey := fmt.Sprintf("products_filtered:%s:%s:%v:%s:%d:%d",
		req.RestaurantID, req.CategoryID, req.AvailableOnly, currentTime, req.Limit, offset)
	var cachedResponse *GetProductsResponse
	if err := s.cache.Get(ctx, cacheKey, &cachedResponse); err == nil {
		return cachedResponse, nil
	}

	// Get products from repository
	products, total, err := s.productRepo.GetByRestaurantCategoryAndTime(
		ctx, req.RestaurantID, categoryID, req.AvailableOnly, currentTime, req.Limit, offset)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	// Build response
	response := &GetProductsResponse{
		Products: products,
		Pagination: PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
		TimeInfo: TimeAvailability{
			CurrentTime:   currentTime,
			CurrentDate:   currentDate,
			IsBusinessDay: isBusinessDay(now),
		},
	}

	// Cache for 5 minutes (shorter than other caches due to time sensitivity)
	s.cache.Set(ctx, cacheKey, response, time.Minute*5)

	return response, nil
}

// Helper function to determine if current day is a business day
func isBusinessDay(t time.Time) bool {
	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

func (s *ProductService) clearProductCache(restaurantID string) {
	// In a real implementation, you'd want to use cache tags or patterns
	// For now, we'll just clear some common cache patterns
	ctx := context.Background()
	for limit := 10; limit <= 50; limit += 10 {
		for offset := 0; offset <= 100; offset += 10 {
			cacheKey := fmt.Sprintf("products:%s:%d:%d", restaurantID, limit, offset)
			s.cache.Delete(ctx, cacheKey)
		}
	}
}

// Category Service
type CategoryService struct {
	categoryRepo repositories.ProductCategoryRepository
	productRepo  repositories.ProductRepository
	cache        *cache.RedisCache
}

func NewCategoryService(
	categoryRepo repositories.ProductCategoryRepository,
	productRepo repositories.ProductRepository,
	cache *cache.RedisCache,
) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		cache:        cache,
	}
}

type CreateCategoryRequest struct {
	Name             string  `json:"name" binding:"required"`
	Description      string  `json:"description"`
	ParentCategoryID *string `json:"parent_category_id,omitempty"`
	SortOrder        int     `json:"sort_order"`
	ImgUrl           string  `json:"img_url,omitempty"`
	VideoUrl         string  `json:"video_url,omitempty"`
}

func (s *CategoryService) CreateCategory(ctx context.Context, restaurantID string, req *CreateCategoryRequest) (*models.ProductCategory, error) {
	var parentCategoryID *primitive.ObjectID
	if req.ParentCategoryID != nil && *req.ParentCategoryID != "" {
		objID, err := primitive.ObjectIDFromHex(*req.ParentCategoryID)
		if err != nil {
			return nil, errors.New("invalid parent category ID")
		}
		parentCategoryID = &objID
	}

	category := &models.ProductCategory{
		RestaurantID:     restaurantID,
		Name:             req.Name,
		Description:      req.Description,
		ParentCategoryID: parentCategoryID,
		SortOrder:        req.SortOrder,
		ImgUrl:           req.ImgUrl,
		VideoUrl:         req.VideoUrl,
		IsActive:         true,
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	// Clear cache
	s.cache.Delete(ctx, "categories:"+restaurantID)

	return category, nil
}

func (s *CategoryService) GetCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]models.ProductCategory, error) {
	// Try cache first
	cacheKey := "categories:" + restaurantID
	var cachedCategories []models.ProductCategory
	if err := s.cache.Get(ctx, cacheKey, &cachedCategories); err == nil {
		return cachedCategories, nil
	}

	// Get from database
	categories, err := s.categoryRepo.GetByRestaurantID(ctx, restaurantID)
	if err != nil {
		return nil, err
	}

	// Cache for 1 hour
	s.cache.Set(ctx, cacheKey, categories, time.Hour)

	return categories, nil
}

// UpdateCategory updates an existing category
func (s *CategoryService) UpdateCategory(ctx context.Context, categoryID, restaurantID string, updates map[string]interface{}) error {
	// Parse category ID
	categoryObjectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid category ID")
	}

	// Get the existing category
	category, err := s.categoryRepo.GetByID(ctx, categoryObjectID)
	if err != nil {
		return errors.New("category not found")
	}

	// Verify ownership
	if category.RestaurantID != restaurantID {
		return errors.New("category does not belong to this restaurant")
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		category.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		category.Description = description
	}
	if sortOrder, ok := updates["sort_order"].(int); ok {
		category.SortOrder = sortOrder
	}
	if imgUrl, ok := updates["img_url"].(string); ok {
		category.ImgUrl = imgUrl
	}
	if videoUrl, ok := updates["video_url"].(string); ok {
		category.VideoUrl = videoUrl
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		category.IsActive = isActive
	}

	// Handle parent category update if provided
	if parentCategoryIDStr, ok := updates["parent_category_id"].(string); ok {
		if parentCategoryIDStr == "" {
			category.ParentCategoryID = nil
		} else {
			parentCategoryID, err := primitive.ObjectIDFromHex(parentCategoryIDStr)
			if err != nil {
				return errors.New("invalid parent category ID")
			}
			category.ParentCategoryID = &parentCategoryID
		}
	}

	// Update in database
	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return err
	}

	// Clear cache
	s.cache.Delete(ctx, "categories:"+restaurantID)

	return nil
}

// DeleteCategory deletes a category if it has no products
func (s *CategoryService) DeleteCategory(ctx context.Context, categoryID, restaurantID string) error {
	// Parse category ID
	categoryObjectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid category ID")
	}

	// Get the category
	category, err := s.categoryRepo.GetByID(ctx, categoryObjectID)
	if err != nil {
		return errors.New("category not found")
	}

	// Verify ownership
	if category.RestaurantID != restaurantID {
		return errors.New("category does not belong to this restaurant")
	}

	// Check if category has products
	products, err := s.productRepo.GetByCategoryID(ctx, categoryObjectID, 1, 0)
	if err != nil {
		return errors.New("failed to check category products")
	}
	if len(products) > 0 {
		return errors.New("cannot delete category with existing products, use delete-with-products endpoint or move products first")
	}

	// Delete category
	if err := s.categoryRepo.Delete(ctx, categoryObjectID); err != nil {
		return err
	}

	// Clear cache
	s.cache.Delete(ctx, "categories:"+restaurantID)

	return nil
}

// DeleteCategoryWithProducts deletes a category and all its products
func (s *CategoryService) DeleteCategoryWithProducts(ctx context.Context, categoryID, restaurantID string) error {
	// Parse category ID
	categoryObjectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid category ID")
	}

	// Get the category
	category, err := s.categoryRepo.GetByID(ctx, categoryObjectID)
	if err != nil {
		return errors.New("category not found")
	}

	// Verify ownership
	if category.RestaurantID != restaurantID {
		return errors.New("category does not belong to this restaurant")
	}

	// Get all products in this category
	// We'll need to page through all products since we need to delete them all
	offset := 0
	limit := 50
	deletedCount := 0

	for {
		products, err := s.productRepo.GetByCategoryID(ctx, categoryObjectID, limit, offset)
		if err != nil {
			return errors.New("failed to retrieve products in category")
		}

		if len(products) == 0 {
			break // No more products
		}

		// Delete each product
		for _, product := range products {
			if err := s.productRepo.Delete(ctx, product.ID); err != nil {
				return errors.New("failed to delete product: " + err.Error())
			}
			deletedCount++
		}

		// If we got fewer products than the limit, we're done
		if len(products) < limit {
			break
		}

		offset += limit
	}

	// Now delete the category
	if err := s.categoryRepo.Delete(ctx, categoryObjectID); err != nil {
		return err
	}

	// Clear caches
	s.cache.Delete(ctx, "categories:"+restaurantID)
	// Clear product caches too - simplified version
	s.clearProductCaches(restaurantID)

	return nil
}

// MoveProductsAndDeleteCategory moves all products from one category to another and deletes the source category
func (s *CategoryService) MoveProductsAndDeleteCategory(ctx context.Context, categoryID, targetCategoryID, restaurantID string) error {
	// Parse category IDs
	sourceObjectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid source category ID")
	}

	targetObjectID, err := primitive.ObjectIDFromHex(targetCategoryID)
	if err != nil {
		return errors.New("invalid target category ID")
	}

	// Get both categories
	sourceCategory, err := s.categoryRepo.GetByID(ctx, sourceObjectID)
	if err != nil {
		return errors.New("source category not found")
	}

	targetCategory, err := s.categoryRepo.GetByID(ctx, targetObjectID)
	if err != nil {
		return errors.New("target category not found")
	}

	// Verify ownership
	if sourceCategory.RestaurantID != restaurantID || targetCategory.RestaurantID != restaurantID {
		return errors.New("categories do not belong to this restaurant")
	}

	// Make sure we're not trying to move to the same category
	if sourceObjectID == targetObjectID {
		return errors.New("source and target categories cannot be the same")
	}

	// Get all products in source category
	offset := 0
	limit := 50
	movedCount := 0

	for {
		products, err := s.productRepo.GetByCategoryID(ctx, sourceObjectID, limit, offset)
		if err != nil {
			return errors.New("failed to retrieve products in category")
		}

		if len(products) == 0 {
			break // No more products
		}

		// Update each product to the new category
		for _, product := range products {
			product.CategoryID = targetObjectID
			if err := s.productRepo.Update(ctx, &product); err != nil {
				return errors.New("failed to update product category: " + err.Error())
			}
			movedCount++
		}

		// If we got fewer products than the limit, we're done
		if len(products) < limit {
			break
		}

		offset += limit
	}

	// Now delete the source category
	if err := s.categoryRepo.Delete(ctx, sourceObjectID); err != nil {
		return err
	}

	// Clear caches
	s.cache.Delete(ctx, "categories:"+restaurantID)
	// Clear product caches too - simplified version
	s.clearProductCaches(restaurantID)

	return nil
}

// Helper method to clear product caches
func (s *CategoryService) clearProductCaches(restaurantID string) {
	ctx := context.Background()
	// Clear simple pattern-based caches
	// In a real implementation, you'd want to use cache tags or more sophisticated invalidation
	for limit := 10; limit <= 50; limit += 10 {
		for offset := 0; offset <= 100; offset += 10 {
			cacheKey := fmt.Sprintf("products:%s:%d:%d", restaurantID, limit, offset)
			s.cache.Delete(ctx, cacheKey)
		}
	}
}
