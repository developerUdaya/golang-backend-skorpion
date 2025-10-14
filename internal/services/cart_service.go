package services

import (
	"context"
	"encoding/json"
	"errors"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/cache"
	"time"

	"github.com/google/uuid"
)

type CartService struct {
	cartRepo    repositories.CartRepository
	productRepo repositories.ProductRepository
	cache       *cache.RedisCache
}

func NewCartService(
	cartRepo repositories.CartRepository,
	productRepo repositories.ProductRepository,
	cache *cache.RedisCache,
) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		cache:       cache,
	}
}

type AddToCartRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
	Price     float64 `json:"price" binding:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gte=0"`
}

type CartResponse struct {
	Cart  *models.Cart       `json:"cart"`
	Items []CartItemResponse `json:"items"`
}

type CartItemResponse struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Total       float64 `json:"total"`
}

func (s *CartService) GetOrCreateCart(ctx context.Context, userID, restaurantID string) (*CartResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	restUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	// Try to get existing active cart
	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil || cart.RestaurantID != restUUID {
		// Create new cart
		cart = &models.Cart{
			UserID:       userUUID,
			RestaurantID: restUUID,
			Items:        models.JSONB{},
			TotalAmount:  0,
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.cartRepo.Create(ctx, cart); err != nil {
			return nil, err
		}
	}

	return s.buildCartResponse(ctx, cart)
}

func (s *CartService) AddToCart(ctx context.Context, userID, restaurantID string, req *AddToCartRequest) (*CartResponse, error) {
	// Get or create cart
	cartResponse, err := s.GetOrCreateCart(ctx, userID, restaurantID)
	if err != nil {
		return nil, err
	}

	cart := cartResponse.Cart

	// Verify product belongs to the same restaurant
	// This would involve getting product from MongoDB and checking restaurant_id
	// For now, we'll assume it's valid

	// Parse existing items
	var items []models.CartItem
	if cart.Items != nil {
		itemsJson, _ := json.Marshal(cart.Items)
		json.Unmarshal(itemsJson, &items)
	}

	// Check if item already exists
	found := false
	for i, item := range items {
		if item.ProductID == req.ProductID {
			items[i].Quantity += req.Quantity
			found = true
			break
		}
	}

	if !found {
		// Add new item
		newItem := models.CartItem{
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			Price:     req.Price,
		}
		items = append(items, newItem)
	}

	// Update cart
	itemsMap := make(map[string]interface{})
	itemsJson, _ := json.Marshal(items)
	json.Unmarshal(itemsJson, &itemsMap)

	cart.Items = itemsMap
	cart.TotalAmount = s.calculateTotal(items)
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return nil, err
	}

	// Clear cache
	s.clearCartCache(userID)

	return s.buildCartResponse(ctx, cart)
}

func (s *CartService) UpdateCartItem(ctx context.Context, userID string, req *UpdateCartItemRequest) (*CartResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	// Parse existing items
	var items []models.CartItem
	if cart.Items != nil {
		itemsJson, _ := json.Marshal(cart.Items)
		json.Unmarshal(itemsJson, &items)
	}

	// Update or remove item
	for i, item := range items {
		if item.ProductID == req.ProductID {
			if req.Quantity == 0 {
				// Remove item
				items = append(items[:i], items[i+1:]...)
			} else {
				// Update quantity
				items[i].Quantity = req.Quantity
			}
			break
		}
	}

	// Update cart
	itemsMap := make(map[string]interface{})
	itemsJson, _ := json.Marshal(items)
	json.Unmarshal(itemsJson, &itemsMap)

	cart.Items = itemsMap
	cart.TotalAmount = s.calculateTotal(items)
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return nil, err
	}

	// Clear cache
	s.clearCartCache(userID)

	return s.buildCartResponse(ctx, cart)
}

func (s *CartService) RemoveFromCart(ctx context.Context, userID, productID string) (*CartResponse, error) {
	req := &UpdateCartItemRequest{
		ProductID: productID,
		Quantity:  0,
	}
	return s.UpdateCartItem(ctx, userID, req)
}

func (s *CartService) ClearCart(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return errors.New("cart not found")
	}

	cart.Items = models.JSONB{}
	cart.TotalAmount = 0
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return err
	}

	// Clear cache
	s.clearCartCache(userID)

	return nil
}

func (s *CartService) GetCart(ctx context.Context, userID string) (*CartResponse, error) {
	// Try cache first
	cacheKey := "cart:" + userID
	var cachedResponse CartResponse
	if err := s.cache.Get(ctx, cacheKey, &cachedResponse); err == nil {
		return &cachedResponse, nil
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	response, err := s.buildCartResponse(ctx, cart)
	if err != nil {
		return nil, err
	}

	// Cache for 10 minutes
	s.cache.Set(ctx, cacheKey, response, time.Minute*10)

	return response, nil
}

func (s *CartService) buildCartResponse(ctx context.Context, cart *models.Cart) (*CartResponse, error) {
	var items []models.CartItem
	if cart.Items != nil {
		itemsJson, _ := json.Marshal(cart.Items)
		json.Unmarshal(itemsJson, &items)
	}

	var itemResponses []CartItemResponse
	for _, item := range items {
		// You could fetch product details here to get product name
		itemResponse := CartItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Total:     item.Price * float64(item.Quantity),
		}
		itemResponses = append(itemResponses, itemResponse)
	}

	return &CartResponse{
		Cart:  cart,
		Items: itemResponses,
	}, nil
}

func (s *CartService) calculateTotal(items []models.CartItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

func (s *CartService) clearCartCache(userID string) {
	ctx := context.Background()
	cacheKey := "cart:" + userID
	s.cache.Delete(ctx, cacheKey)
}

// ApplyCoupon applies a coupon to the user's cart
func (s *CartService) ApplyCoupon(ctx context.Context, userID, couponCode string) (*CartResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	// TODO: Validate coupon code and get coupon ID from coupon service
	// For now, we'll generate a mock coupon ID
	couponID := uuid.New()
	cart.CouponID = &couponID
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return nil, err
	}

	// Clear cache
	s.clearCartCache(userID)

	return s.buildCartResponse(ctx, cart)
}

// RemoveCoupon removes the applied coupon from the user's cart
func (s *CartService) RemoveCoupon(ctx context.Context, userID string) (*CartResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	cart.CouponID = nil
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return nil, err
	}

	// Clear cache
	s.clearCartCache(userID)

	return s.buildCartResponse(ctx, cart)
}
