package handlers

import (
	"context"
	"golang-food-backend/internal/services"
)

// CartServiceInterface defines the contract for cart service
type CartServiceInterface interface {
	GetOrCreateCart(ctx context.Context, userID, restaurantID string) (*services.CartResponse, error)
	AddToCart(ctx context.Context, userID, restaurantID string, req *services.AddToCartRequest) (*services.CartResponse, error)
	UpdateCartItem(ctx context.Context, userID string, req *services.UpdateCartItemRequest) (*services.CartResponse, error)
	RemoveFromCart(ctx context.Context, userID, productID string) (*services.CartResponse, error)
	ClearCart(ctx context.Context, userID string) error
	ApplyCoupon(ctx context.Context, userID, couponCode string) (*services.CartResponse, error)
	RemoveCoupon(ctx context.Context, userID string) (*services.CartResponse, error)
	GetBillSummary(ctx context.Context, userID, restaurantID, addressID string) (*services.BillSummaryResponse, error)
	Checkout(ctx context.Context, userID, restaurantID, addressID string) (*services.CheckoutResponse, error)
}
