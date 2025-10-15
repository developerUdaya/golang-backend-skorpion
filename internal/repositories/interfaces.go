package repositories

import (
	"context"
	"golang-food-backend/internal/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository interface for PostgreSQL user operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByPhone(ctx context.Context, phone string) (*models.User, error)
	// Multi-restaurant support: get user by email within specific restaurant
	GetByEmailAndRestaurant(ctx context.Context, email string, restaurantID uuid.UUID) (*models.User, error)
	// Multi-restaurant support: get user by phone within specific restaurant
	GetByPhoneAndRestaurant(ctx context.Context, phone string, restaurantID uuid.UUID) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID) ([]models.User, error)
}

// OTPRepository interface for PostgreSQL OTP operations
type OTPRepository interface {
	Create(ctx context.Context, otp *models.OTP) error
	GetValidOTP(ctx context.Context, phone string, restaurantID uuid.UUID, otpCode string) (*models.OTP, error)
	GetValidOTPWithOptionalRestaurant(ctx context.Context, phone string, restaurantID *uuid.UUID, otpCode string) (*models.OTP, error)
	InvalidateOTP(ctx context.Context, id uuid.UUID) error
	DeleteExpiredOTPs(ctx context.Context) error
	IncrementAttempt(ctx context.Context, id uuid.UUID) error
}

// RestaurantRepository interface for PostgreSQL restaurant operations
type RestaurantRepository interface {
	Create(ctx context.Context, restaurant *models.Restaurant) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Restaurant, error)
	Update(ctx context.Context, restaurant *models.Restaurant) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]models.Restaurant, error)
	Search(ctx context.Context, query string, limit, offset int) ([]models.Restaurant, error)
	GetRestaurantsWithAutoOpenClose() ([]*models.Restaurant, error)
}

// OrderRepository interface for PostgreSQL order operations
type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	Update(ctx context.Context, order *models.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error)
	GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]models.Order, error)
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.Order, error)
}

// PaymentRepository interface for PostgreSQL payment operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Payment, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Payment, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error)
	Update(ctx context.Context, payment *models.Payment) error
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.Payment, error)
}

// CartRepository interface for PostgreSQL cart operations
type CartRepository interface {
	Create(ctx context.Context, cart *models.Cart) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Cart, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Cart, error)
	Update(ctx context.Context, cart *models.Cart) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProductRepository interface for MongoDB product operations
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	GetByRestaurantID(ctx context.Context, restaurantID string, limit, offset int) ([]models.Product, error)
	GetByCategoryID(ctx context.Context, categoryID primitive.ObjectID, limit, offset int) ([]models.Product, error)
	Search(ctx context.Context, query string, restaurantID string, limit, offset int) ([]models.Product, error)
	GetHighlighted(ctx context.Context, restaurantID string, highlightType string) ([]models.Product, error)
	GetByRestaurantCategoryAndTime(ctx context.Context, restaurantID string, categoryID *primitive.ObjectID, availableOnly bool, currentTime string, limit, offset int) ([]models.Product, int64, error)
}

// ProductCategoryRepository interface for MongoDB category operations
type ProductCategoryRepository interface {
	Create(ctx context.Context, category *models.ProductCategory) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.ProductCategory, error)
	Update(ctx context.Context, category *models.ProductCategory) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	GetByRestaurantID(ctx context.Context, restaurantID string) ([]models.ProductCategory, error)
}

// RatingReviewRepository interface for MongoDB review operations
type RatingReviewRepository interface {
	Create(ctx context.Context, review *models.RatingReview) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.RatingReview, error)
	Update(ctx context.Context, review *models.RatingReview) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	GetByEntityID(ctx context.Context, entityID string, reviewType string, limit, offset int) ([]models.RatingReview, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]models.RatingReview, error)
}

// InventoryRepository interface for MongoDB inventory operations
type InventoryRepository interface {
	Create(ctx context.Context, inventory *models.Inventory) error
	GetByProductID(ctx context.Context, productID primitive.ObjectID) (*models.Inventory, error)
	Update(ctx context.Context, inventory *models.Inventory) error
	UpdateQuantity(ctx context.Context, productID primitive.ObjectID, quantity int) error
	ReserveStock(ctx context.Context, productID primitive.ObjectID, quantity int) error
	ReleaseStock(ctx context.Context, productID primitive.ObjectID, quantity int) error
	GetLowStock(ctx context.Context, restaurantID string) ([]models.Inventory, error)
}

// CouponRepository interface for PostgreSQL coupon operations
type CouponRepository interface {
	Create(ctx context.Context, coupon *models.Coupon) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Coupon, error)
	GetByCode(ctx context.Context, code string) (*models.Coupon, error)
	Update(ctx context.Context, coupon *models.Coupon) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetCouponsWithFilters(ctx context.Context, offset, limit int, restaurantID *uuid.UUID, active *bool) ([]models.Coupon, int64, error)
}

// RefundRepository interface for PostgreSQL refund operations
type RefundRepository interface {
	Create(ctx context.Context, refund *models.Refund) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Refund, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Refund, error)
	Update(ctx context.Context, refund *models.Refund) error
	GetByUserIDWithFilters(ctx context.Context, userID uuid.UUID, offset, limit int, status string) ([]models.Refund, int64, error)
}

// AddressRepository interface for PostgreSQL address operations
type AddressRepository interface {
	Create(ctx context.Context, address *models.Address) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Address, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Address, int64, error)
	Update(ctx context.Context, address *models.Address) error
	Delete(ctx context.Context, id uuid.UUID) error
	UnsetDefaultAddresses(ctx context.Context, userID uuid.UUID) error
}

// DeliveryPartnerRepository interface for PostgreSQL delivery partner operations
type DeliveryPartnerRepository interface {
	Create(ctx context.Context, partner *models.DeliveryPartnerCompany) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryPartnerCompany, error)
	Update(ctx context.Context, partner *models.DeliveryPartnerCompany) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context, limit, offset int) ([]models.DeliveryPartnerCompany, error)
	GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.DeliveryPartnerCompany, error)
}

// RestaurantDeliveryPartnerRepository interface for restaurant-delivery partner relationships
type RestaurantDeliveryPartnerRepository interface {
	Create(ctx context.Context, relationship *models.RestaurantDeliveryPartners) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.RestaurantDeliveryPartners, error)
	GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID) ([]models.RestaurantDeliveryPartners, error)
	GetByDeliveryPartnerID(ctx context.Context, partnerID uuid.UUID) ([]models.RestaurantDeliveryPartners, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// TimeRangeProductRepository interface for MongoDB time-based product operations
type TimeRangeProductRepository interface {
	CreateTimeGroup(ctx context.Context, group *models.TimeRangeProductsGroup) error
	GetTimeGroupByID(ctx context.Context, id primitive.ObjectID) (*models.TimeRangeProductsGroup, error)
	GetTimeGroupsByRestaurant(ctx context.Context, restaurantID string) ([]models.TimeRangeProductsGroup, error)
	UpdateTimeGroup(ctx context.Context, group *models.TimeRangeProductsGroup) error
	DeleteTimeGroup(ctx context.Context, id primitive.ObjectID) error

	AddProductToTimeGroup(ctx context.Context, item *models.TimeRangeProductsGroupItem) error
	RemoveProductFromTimeGroup(ctx context.Context, groupID, productID primitive.ObjectID) error
	GetProductsByTimeGroup(ctx context.Context, groupID primitive.ObjectID) ([]models.TimeRangeProductsGroupItem, error)
	GetActiveProductsByTime(ctx context.Context, restaurantID string, currentTime string) ([]primitive.ObjectID, error)
}

// PorterDeliveryRepository interface for PostgreSQL Porter delivery operations
type PorterDeliveryRepository interface {
	Create(ctx context.Context, delivery *models.PorterDelivery) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PorterDelivery, error)
	GetByPorterOrderID(ctx context.Context, porterOrderID string) (*models.PorterDelivery, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.PorterDelivery, error)
	GetActiveByOrderID(ctx context.Context, orderID uuid.UUID) (*models.PorterDelivery, error)
	Update(ctx context.Context, delivery *models.PorterDelivery) error
	UpdateStatus(ctx context.Context, porterOrderID string, status string, metadata map[string]interface{}) error
	DeactivateOldDeliveries(ctx context.Context, orderID uuid.UUID) error
}
