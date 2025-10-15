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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CartService struct {
	cartRepo    repositories.CartRepository
	productRepo repositories.ProductRepository
	orderRepo   repositories.OrderRepository
	paymentRepo repositories.PaymentRepository
	cache       *cache.RedisCache
}

func NewCartService(
	cartRepo repositories.CartRepository,
	productRepo repositories.ProductRepository,
	orderRepo repositories.OrderRepository,
	paymentRepo repositories.PaymentRepository,
	cache *cache.RedisCache,
) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		cache:       cache,
	}
}

type AddToCartRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
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

type BillSummaryRequest struct {
	UserID       string `json:"user_id"`
	RestaurantID string `json:"restaurant_id"`
	AddressID    string `json:"address_id"`
}

type BillSummaryResponse struct {
	SubTotal       float64            `json:"sub_total"`
	CouponDetails  *CouponDetails     `json:"coupon_details,omitempty"`
	DeliveryCharge float64            `json:"delivery_charge"`
	TaxAmount      float64            `json:"tax_amount"`
	PackagingFee   float64            `json:"packaging_fee"`
	TotalAmount    float64            `json:"total_amount"`
	Items          []CartItemResponse `json:"items"`
}

type CouponDetails struct {
	CouponCode     string  `json:"coupon_code"`
	DiscountType   string  `json:"discount_type"` // percentage, fixed
	DiscountValue  float64 `json:"discount_value"`
	MaxDiscount    float64 `json:"max_discount,omitempty"`
	DiscountAmount float64 `json:"discount_amount"`
}

type CheckoutRequest struct {
	UserID       string `json:"user_id"`
	RestaurantID string `json:"restaurant_id"`
	AddressID    string `json:"address_id"`
}

type CheckoutResponse struct {
	OrderID       string  `json:"order_id"`
	PaymentID     string  `json:"payment_id"`
	TotalAmount   float64 `json:"total_amount"`
	PaymentMethod string  `json:"payment_method"`
	Status        string  `json:"status"`
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
		}
		items = append(items, newItem)
	}

	// Update cart
	itemsMap := make(map[string]interface{})
	itemsJson, _ := json.Marshal(items)
	json.Unmarshal(itemsJson, &itemsMap)

	cart.Items = itemsMap
	// TotalAmount will be calculated in buildCartResponse with current prices
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
	// TotalAmount will be calculated in buildCartResponse with current prices
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
	var total float64

	for _, item := range items {
		// Fetch current product data from MongoDB to get current price
		productID, err := primitive.ObjectIDFromHex(item.ProductID)
		if err != nil {
			continue // Invalid product ID, skip
		}

		product, err := s.productRepo.GetByID(ctx, productID)
		if err != nil {
			// If product not found, skip this item
			continue
		}

		currentPrice := product.Price
		if product.DiscountPrice != nil && *product.DiscountPrice > 0 {
			currentPrice = *product.DiscountPrice
		}

		itemResponse := CartItemResponse{
			ProductID:   item.ProductID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
			Price:       currentPrice,
			Total:       currentPrice * float64(item.Quantity),
		}
		itemResponses = append(itemResponses, itemResponse)
		total += currentPrice * float64(item.Quantity)
	}

	// Update cart total amount with current prices
	cart.TotalAmount = total
	s.cartRepo.Update(ctx, cart)

	return &CartResponse{
		Cart:  cart,
		Items: itemResponses,
	}, nil
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

// GetBillSummary calculates the complete bill summary including taxes, delivery charges, etc.
func (s *CartService) GetBillSummary(ctx context.Context, userID, restaurantID, addressID string) (*BillSummaryResponse, error) {
	// Get user's cart
	cartResponse, err := s.GetOrCreateCart(ctx, userID, restaurantID)
	if err != nil {
		return nil, err
	}

	if len(cartResponse.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	// Calculate subtotal from cart items
	var subTotal float64
	for _, item := range cartResponse.Items {
		subTotal += item.Total
	}

	// TODO: Get restaurant details to calculate packaging fee and tax rate
	// For now using default values
	packagingFee := 10.0 // ₹10 packaging fee
	taxRate := 0.18      // 18% GST
	taxAmount := subTotal * taxRate

	// TODO: Calculate delivery charge using Porter API or restaurant's delivery partner
	deliveryCharge := 30.0 // Default ₹30 delivery charge

	// TODO: Get and apply coupon if exists
	var couponDetails *CouponDetails
	var couponDiscount float64 = 0

	if cartResponse.Cart.CouponID != nil {
		// TODO: Fetch coupon details and calculate discount
		couponDetails = &CouponDetails{
			CouponCode:     "SAVE20",
			DiscountType:   "percentage",
			DiscountValue:  20,
			DiscountAmount: subTotal * 0.2,
		}
		couponDiscount = couponDetails.DiscountAmount
	}

	totalAmount := subTotal + taxAmount + packagingFee + deliveryCharge - couponDiscount

	return &BillSummaryResponse{
		SubTotal:       subTotal,
		CouponDetails:  couponDetails,
		DeliveryCharge: deliveryCharge,
		TaxAmount:      taxAmount,
		PackagingFee:   packagingFee,
		TotalAmount:    totalAmount,
		Items:          cartResponse.Items,
	}, nil
}

// Checkout processes the cart and creates order and payment records
func (s *CartService) Checkout(ctx context.Context, userID, restaurantID, addressID string) (*CheckoutResponse, error) {
	// Get bill summary first to calculate total amount
	billSummary, err := s.GetBillSummary(ctx, userID, restaurantID, addressID)
	if err != nil {
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	restUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	addressUUID, err := uuid.Parse(addressID)
	if err != nil {
		return nil, errors.New("invalid address ID")
	}

	// Get user's cart
	cart, err := s.cartRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	// Create order
	order := &models.Order{
		UserID:          userUUID,
		RestaurantID:    restUUID,
		CartID:          cart.ID,
		OrderStatus:     "pending",
		AddressID:       &addressUUID,
		TotalAmount:     billSummary.TotalAmount,
		CreatedAt:       time.Now(),
		DiscountDetails: models.JSONB{},
		OrderLogs:       models.JSONB{},
	}

	// Add discount details if coupon was applied
	if billSummary.CouponDetails != nil {
		discountData := map[string]interface{}{
			"coupon_code":     billSummary.CouponDetails.CouponCode,
			"discount_type":   billSummary.CouponDetails.DiscountType,
			"discount_value":  billSummary.CouponDetails.DiscountValue,
			"discount_amount": billSummary.CouponDetails.DiscountAmount,
		}
		order.DiscountDetails = discountData
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Create payment record
	payment := &models.Payment{
		OrderID:   order.ID,
		UserID:    userUUID,
		Amount:    billSummary.TotalAmount,
		Method:    "razorpay", // Default to Razorpay
		Status:    "pending",
		CreatedAt: time.Now(),
		Metadata:  models.JSONB{},
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Update order with payment ID
	order.PaymentID = &payment.ID
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	return &CheckoutResponse{
		OrderID:       order.ID.String(),
		PaymentID:     payment.ID.String(),
		TotalAmount:   billSummary.TotalAmount,
		PaymentMethod: "razorpay",
		Status:        "pending",
	}, nil
}
