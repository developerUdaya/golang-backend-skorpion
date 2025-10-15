package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONB type for PostgreSQL
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// StringArray type for PostgreSQL arrays
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, s)
}

// User model - PostgreSQL (strict, consistent data)
// Multi-restaurant support:
// - For customers: same email/phone can exist across different restaurants
// - For other roles (admin, restaurant_owner, restaurant_staff): email/phone must be globally unique
type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name             string     `gorm:"not null" json:"name"`
	Email            string     `gorm:"not null" json:"email"`
	Phone            string     `gorm:"not null" json:"phone"`
	PasswordHash     string     `gorm:"not null" json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DefaultAddressID *uuid.UUID `gorm:"type:uuid" json:"default_address_id"`
	WalletBalance    float64    `gorm:"default:0" json:"wallet_balance"`
	IsVerified       bool       `gorm:"default:false" json:"is_verified"`
	Status           string     `gorm:"default:active" json:"status"`   // active, inactive, suspended
	RestaurantID     *uuid.UUID `gorm:"type:uuid" json:"restaurant_id"` // Required for customers/restaurant staff, optional for admins
	Role             string     `gorm:"default:customer" json:"role"`   // customer, restaurant_owner, restaurant_staff, admin

	// Database constraints will be:
	// 1. Unique index on (email, restaurant_id) for customers
	// 2. Unique index on (phone, restaurant_id) for customers
	// 3. Unique index on email for non-customer roles
	// 4. Unique index on phone for non-customer roles
}

// Restaurant model - PostgreSQL
type Restaurant struct {
	ID                uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name              string      `gorm:"not null" json:"name"`
	Description       string      `json:"description"`
	Logo              string      `json:"logo"`
	CuisineTypes      StringArray `gorm:"type:jsonb" json:"cuisine_types"`
	OwnerID           uuid.UUID   `gorm:"type:uuid;not null" json:"owner_id"`
	Owner             User        `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	GSTNumber         string      `gorm:"uniqueIndex" json:"gst_number"`
	Status            string      `gorm:"default:active" json:"status"`               // active, inactive, suspended, closed
	IsOpen            bool        `gorm:"default:true" json:"is_open"`                // real-time open/closed status
	OpeningHours      JSONB       `gorm:"type:jsonb" json:"opening_hours"`            // shop timing for each day
	AutoOpenClose     bool        `gorm:"default:true" json:"auto_open_close"`        // auto manage open/close based on timing
	TimeZone          string      `gorm:"default:'Asia/Kolkata'" json:"timezone"`     // restaurant timezone
	LastStatusUpdate  *time.Time  `json:"last_status_update"`                         // when status was last updated
	PreparationTime   int         `gorm:"default:15" json:"preparation_time_minutes"` // average prep time
	CreatedAt         time.Time   `json:"created_at"`
	PickupLocationID  *uuid.UUID  `gorm:"type:uuid" json:"pickup_location_id"`
	FranchiseParentID *uuid.UUID  `gorm:"type:uuid" json:"franchise_parent_id"`
	ContactNumber     string      `json:"contact_number"`
}

// RestaurantDeliveryPartners model - PostgreSQL
type RestaurantDeliveryPartners struct {
	ID                       uuid.UUID              `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RestaurantID             uuid.UUID              `gorm:"type:uuid;not null" json:"restaurant_id"`
	Restaurant               Restaurant             `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	DeliveryPartnerCompanyID uuid.UUID              `gorm:"type:uuid;not null" json:"delivery_partner_company_id"`
	DeliveryPartnerCompany   DeliveryPartnerCompany `gorm:"foreignKey:DeliveryPartnerCompanyID" json:"delivery_partner_company,omitempty"`
}

// RestaurantDeliveryLocationBoundary model - PostgreSQL
type RestaurantDeliveryLocationBoundary struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RestaurantID     uuid.UUID  `gorm:"type:uuid;not null" json:"restaurant_id"`
	Restaurant       Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	GeoPolygon       JSONB      `gorm:"type:jsonb" json:"geo_polygon"`
	DeliveryRadiusKm float64    `json:"delivery_radius_km"`
	MinOrderValue    float64    `json:"min_order_value"`
	DeliveryFee      float64    `json:"delivery_fee"`
}

// CommissionToRestaurant model - PostgreSQL
type CommissionToRestaurant struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RestaurantID    uuid.UUID  `gorm:"type:uuid;not null" json:"restaurant_id"`
	Restaurant      Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	CommissionType  string     `gorm:"not null" json:"commission_type"` // percentage, flat
	CommissionValue float64    `gorm:"not null" json:"commission_value"`
}

// Cart model - PostgreSQL (transactional data)
type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Cart struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	User         User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Items        JSONB      `gorm:"type:jsonb" json:"items"`
	RestaurantID uuid.UUID  `gorm:"type:uuid;not null" json:"restaurant_id"`
	Restaurant   Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	CouponID     *uuid.UUID `gorm:"type:uuid" json:"coupon_id"`
	TotalAmount  float64    `json:"total_amount"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Status       string     `gorm:"default:active" json:"status"`
}

// Order model - PostgreSQL (critical transactional data)
type Order struct {
	ID                             uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID                         uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	User                           User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RestaurantID                   uuid.UUID        `gorm:"type:uuid;not null" json:"restaurant_id"`
	Restaurant                     Restaurant       `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	CartID                         uuid.UUID        `gorm:"type:uuid;not null" json:"cart_id"`
	Cart                           Cart             `gorm:"foreignKey:CartID" json:"cart,omitempty"`
	OrderStatus                    string           `gorm:"default:pending" json:"order_status"` // pending, confirmed, preparing, dispatched, delivered, cancelled
	DeliveryPartnerID              *uuid.UUID       `gorm:"type:uuid" json:"delivery_partner_id"`
	PaymentID                      *uuid.UUID       `gorm:"type:uuid" json:"payment_id"`
	AddressID                      *uuid.UUID       `gorm:"type:uuid" json:"address_id"`
	OrderLogs                      JSONB            `gorm:"type:jsonb" json:"order_logs"`
	TotalAmount                    float64          `json:"total_amount"`
	CreatedAt                      time.Time        `json:"created_at"`
	DiscountDetails                JSONB            `gorm:"type:jsonb" json:"discount_details"`
	PickupFullAddressWithLatLong   JSONB            `gorm:"type:jsonb" json:"pickup_full_address_with_lat_long"`
	DeliveryFullAddressWithLatLong JSONB            `gorm:"type:jsonb" json:"delivery_full_address_with_lat_long"`
	CustomerName                   string           `json:"customer_name"`
	CustomerContact                string           `json:"customer_contact"`
	PorterDeliveries               []PorterDelivery `gorm:"foreignKey:OrderID" json:"porter_deliveries,omitempty"`
	ActivePorterDeliveryID         *uuid.UUID       `gorm:"type:uuid" json:"active_porter_delivery_id"`
}

// PorterDelivery model - PostgreSQL (tracks Porter delivery details for orders)
type PorterDelivery struct {
	ID                    uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID               uuid.UUID  `gorm:"type:uuid;not null" json:"order_id"`
	Order                 Order      `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	PorterOrderID         string     `gorm:"not null;uniqueIndex" json:"porter_order_id"` // Porter's order ID
	Status                string     `gorm:"default:created" json:"status"`               // created, assigned, picked_up, in_transit, delivered, cancelled, failed
	PartnerName           string     `json:"partner_name"`
	PartnerPhoneNumber    string     `json:"partner_phone_number"`
	PartnerPicture        string     `json:"partner_picture"`
	VehicleType           string     `json:"vehicle_type"`
	VehicleNumber         string     `json:"vehicle_number"`
	TrackingURL           string     `json:"tracking_url"`
	EstimatedDeliveryTime *time.Time `json:"estimated_delivery_time"`
	ActualDeliveryTime    *time.Time `json:"actual_delivery_time"`
	PickupTime            *time.Time `json:"pickup_time"`
	DeliveryFee           float64    `json:"delivery_fee"`
	Distance              float64    `json:"distance"` // in kilometers
	IsActive              bool       `gorm:"default:true" json:"is_active"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	PorterResponse        JSONB      `gorm:"type:jsonb" json:"porter_response"` // Full Porter API response
	StatusHistory         JSONB      `gorm:"type:jsonb" json:"status_history"`  // Track all status changes
}

// OrderLog model - PostgreSQL
type OrderLog struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	Order     Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Status    string    `gorm:"not null" json:"status"`
	Note      string    `json:"note"`
	Timestamp time.Time `gorm:"default:now()" json:"timestamp"`
}

// Payment model - PostgreSQL (critical financial data)
type Payment struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID       uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	Order         Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	UserID        uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount        float64   `gorm:"not null" json:"amount"`
	Method        string    `gorm:"not null" json:"method"`        // UPI, card, wallet, cash
	Status        string    `gorm:"default:pending" json:"status"` // pending, success, failed
	TransactionID string    `gorm:"uniqueIndex" json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
	Metadata      JSONB     `gorm:"type:jsonb" json:"metadata"`
}

// Refund model - PostgreSQL
type Refund struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID      uuid.UUID  `gorm:"type:uuid;not null" json:"order_id"`
	Order        Order      `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	PaymentID    uuid.UUID  `gorm:"type:uuid;not null" json:"payment_id"`
	Payment      Payment    `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	User         User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Amount       float64    `gorm:"not null" json:"amount"`
	Reason       string     `json:"reason"`
	Status       string     `gorm:"default:pending" json:"status"` // pending, approved, rejected, processed, failed
	AdminComment *string    `json:"admin_comment"`
	ProcessedAt  *time.Time `json:"processed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DeliveryPartnerCompany model - PostgreSQL
type DeliveryPartnerCompany struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	APIKey      string    `gorm:"uniqueIndex;not null" json:"api_key"`
	ContactInfo JSONB     `gorm:"type:jsonb" json:"contact_info"`
	GSTNumber   string    `gorm:"uniqueIndex" json:"gst_number"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `gorm:"default:active" json:"status"`
}

// Favourite model - PostgreSQL
type Favourite struct {
	ID           uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID       uuid.UUID   `gorm:"type:uuid;not null" json:"user_id"`
	User         User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	RestaurantID *uuid.UUID  `gorm:"type:uuid" json:"restaurant_id"`
	Restaurant   *Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	ProductID    *string     `json:"product_id"` // MongoDB reference
	CreatedAt    time.Time   `json:"created_at"`
}

// Coupon model - PostgreSQL
type Coupon struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Code          string     `gorm:"uniqueIndex;not null" json:"code"`
	Description   string     `json:"description"`
	DiscountType  string     `gorm:"not null" json:"discount_type"` // flat, percentage
	DiscountValue float64    `gorm:"not null" json:"discount_value"`
	MaxDiscount   float64    `json:"max_discount"`
	MinOrderValue float64    `json:"min_order_value"`
	ValidFrom     time.Time  `json:"valid_from"`
	ValidTo       time.Time  `json:"valid_to"`
	UsageLimit    int        `gorm:"default:-1" json:"usage_limit"` // -1 for unlimited
	UsedCount     int        `gorm:"default:0" json:"used_count"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	RestaurantID  *uuid.UUID `gorm:"type:uuid" json:"restaurant_id"` // null for platform-wide coupons
}

// Notification model - PostgreSQL
type Notification struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User     User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type     string    `gorm:"not null" json:"type"` // push, email, sms
	Title    string    `gorm:"not null" json:"title"`
	Message  string    `gorm:"not null" json:"message"`
	Metadata JSONB     `gorm:"type:jsonb" json:"metadata"`
	SentAt   time.Time `gorm:"default:now()" json:"sent_at"`
	Status   string    `gorm:"default:pending" json:"status"` // pending, sent, failed
}

// AdminUser model - PostgreSQL
type AdminUser struct {
	ID           uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string      `gorm:"not null" json:"name"`
	Email        string      `gorm:"uniqueIndex;not null" json:"email"`
	Role         string      `gorm:"not null" json:"role"`
	Permissions  StringArray `gorm:"type:jsonb" json:"permissions"`
	PasswordHash string      `gorm:"not null" json:"-"`
	CreatedAt    time.Time   `json:"created_at"`
	IsActive     bool        `gorm:"default:true" json:"is_active"`
}

// AuditLog model - PostgreSQL
type AuditLog struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	EntityType  string    `gorm:"not null" json:"entity_type"`
	EntityID    string    `gorm:"not null" json:"entity_id"`
	Action      string    `gorm:"not null" json:"action"`
	PerformedBy uuid.UUID `gorm:"type:uuid;not null" json:"performed_by"`
	Timestamp   time.Time `gorm:"default:now()" json:"timestamp"`
	Metadata    JSONB     `gorm:"type:jsonb" json:"metadata"`
}

// Address model - PostgreSQL (user addresses)
type Address struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type         string    `gorm:"not null" json:"type"` // home, office, other
	AddressLine1 string    `gorm:"not null" json:"address_line1"`
	AddressLine2 string    `json:"address_line2"`
	City         string    `gorm:"not null" json:"city"`
	State        string    `gorm:"not null" json:"state"`
	Country      string    `gorm:"not null" json:"country"`
	PinCode      string    `gorm:"not null" json:"pin_code"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OTP model for SMS authentication
type OTP struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Phone        string     `gorm:"not null" json:"phone"`
	RestaurantID *uuid.UUID `gorm:"type:uuid" json:"restaurant_id"` // OTP can be restaurant-specific for customers, NULL for admin/staff
	OTPCode      string     `gorm:"not null" json:"-"`              // Don't expose in JSON
	ExpiresAt    time.Time  `gorm:"not null" json:"expires_at"`
	IsUsed       bool       `gorm:"default:false" json:"is_used"`
	AttemptCount int        `gorm:"default:0" json:"attempt_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
