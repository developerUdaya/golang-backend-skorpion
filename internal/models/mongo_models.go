package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product model - MongoDB (flexible catalog data)
type Product struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	RestaurantID    string                 `bson:"restaurant_id" json:"restaurant_id"`
	CategoryID      primitive.ObjectID     `bson:"category_id" json:"category_id"`
	Name            string                 `bson:"name" json:"name"`
	Description     string                 `bson:"description" json:"description"`
	Price           float64                `bson:"price" json:"price"`
	DiscountPrice   *float64               `bson:"discount_price,omitempty" json:"discount_price"`
	ImageUrls       []string               `bson:"image_urls" json:"image_urls"`
	IsAvailable     bool                   `bson:"is_available" json:"is_available"`
	PreparationTime int                    `bson:"preparation_time" json:"preparation_time"` // in minutes
	Tags            []string               `bson:"tags" json:"tags"`
	CreatedAt       time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time              `bson:"updated_at" json:"updated_at"`
	VideoUrl        string                 `bson:"video_url,omitempty" json:"video_url"`
	NutritionalInfo map[string]interface{} `bson:"nutritional_info,omitempty" json:"nutritional_info"`
	Variants        []ProductVariant       `bson:"variants,omitempty" json:"variants"`
	Addons          []ProductAddon         `bson:"addons,omitempty" json:"addons"`
}

// ProductVariant for size/type variations
type ProductVariant struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Price float64            `bson:"price" json:"price"`
}

// ProductAddon for extra items
type ProductAddon struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Price float64            `bson:"price" json:"price"`
}

// ProductCategory model - MongoDB
type ProductCategory struct {
	ID               primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	RestaurantID     string              `bson:"restaurant_id" json:"restaurant_id"`
	Name             string              `bson:"name" json:"name"`
	Description      string              `bson:"description" json:"description"`
	ParentCategoryID *primitive.ObjectID `bson:"parent_category_id,omitempty" json:"parent_category_id"`
	SortOrder        int                 `bson:"sort_order" json:"sort_order"`
	ImgUrl           string              `bson:"img_url,omitempty" json:"img_url"`
	VideoUrl         string              `bson:"video_url,omitempty" json:"video_url"`
	IsActive         bool                `bson:"is_active" json:"is_active"`
	CreatedAt        time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
}

// HighlightProduct model - MongoDB
type HighlightProduct struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID     primitive.ObjectID `bson:"product_id" json:"product_id"`
	RestaurantID  string             `bson:"restaurant_id" json:"restaurant_id"`
	HighlightType string             `bson:"highlight_type" json:"highlight_type"` // featured, bestseller, recommended
	StartDate     time.Time          `bson:"start_date" json:"start_date"`
	EndDate       time.Time          `bson:"end_date" json:"end_date"`
	IsActive      bool               `bson:"is_active" json:"is_active"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

// TimeRangeProductsGroup model - MongoDB
type TimeRangeProductsGroup struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID string             `bson:"restaurant_id" json:"restaurant_id"`
	GroupName    string             `bson:"group_name" json:"group_name"`
	StartTime    string             `bson:"start_time" json:"start_time"` // HH:MM format
	EndTime      string             `bson:"end_time" json:"end_time"`     // HH:MM format
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// TimeRangeProductsGroupItem model - MongoDB
type TimeRangeProductsGroupItem struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID primitive.ObjectID `bson:"product_id" json:"product_id"`
	GroupID   primitive.ObjectID `bson:"group_id" json:"group_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// RatingReview model - MongoDB (reviews are flexible data)
type RatingReview struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         string             `bson:"user_id" json:"user_id"`
	OrderID        string             `bson:"order_id" json:"order_id"`
	ReviewType     string             `bson:"review_type" json:"review_type"` // restaurant, product, delivery_partner
	EntityID       string             `bson:"entity_id" json:"entity_id"`     // restaurant_id, product_id, or partner_id
	Rating         int                `bson:"rating" json:"rating"`           // 1-5
	ReviewText     string             `bson:"review_text" json:"review_text"`
	Images         []string           `bson:"images,omitempty" json:"images"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	IsVerified     bool               `bson:"is_verified" json:"is_verified"`
	HelpfulCount   int                `bson:"helpful_count" json:"helpful_count"`
	ReplyFromOwner *OwnerReply        `bson:"reply_from_owner,omitempty" json:"reply_from_owner"`
}

type OwnerReply struct {
	Reply     string    `bson:"reply" json:"reply"`
	RepliedAt time.Time `bson:"replied_at" json:"replied_at"`
}

// Banner model - MongoDB (marketing content)
type Banner struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	ImageUrl        string             `bson:"image_url" json:"image_url"`
	TargetUrl       string             `bson:"target_url,omitempty" json:"target_url"`
	DisplayLocation string             `bson:"display_location" json:"display_location"` // home, offers, restaurant
	StartDate       time.Time          `bson:"start_date" json:"start_date"`
	EndDate         time.Time          `bson:"end_date" json:"end_date"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	RestaurantID    string             `bson:"restaurant_id,omitempty" json:"restaurant_id"` // null for platform banners
	Priority        int                `bson:"priority" json:"priority"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// SystemLog model - MongoDB (flexible logging)
type SystemLog struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Level     string                 `bson:"level" json:"level"` // info, warning, error
	Service   string                 `bson:"service" json:"service"`
	Message   string                 `bson:"message" json:"message"`
	Data      map[string]interface{} `bson:"data,omitempty" json:"data"`
	UserID    string                 `bson:"user_id,omitempty" json:"user_id"`
	IP        string                 `bson:"ip,omitempty" json:"ip"`
	UserAgent string                 `bson:"user_agent,omitempty" json:"user_agent"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
}

// SearchLog model - MongoDB (for analytics)
type SearchLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       string             `bson:"user_id,omitempty" json:"user_id"`
	RestaurantID string             `bson:"restaurant_id,omitempty" json:"restaurant_id"`
	Query        string             `bson:"query" json:"query"`
	ResultsCount int                `bson:"results_count" json:"results_count"`
	ClickedItems []string           `bson:"clicked_items,omitempty" json:"clicked_items"`
	Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
	IP           string             `bson:"ip,omitempty" json:"ip"`
}

// UserActivity model - MongoDB (user behavior tracking)
type UserActivity struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID       string                 `bson:"user_id" json:"user_id"`
	RestaurantID string                 `bson:"restaurant_id,omitempty" json:"restaurant_id"`
	ActivityType string                 `bson:"activity_type" json:"activity_type"` // view_product, add_to_cart, place_order
	EntityID     string                 `bson:"entity_id,omitempty" json:"entity_id"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty" json:"metadata"`
	Timestamp    time.Time              `bson:"timestamp" json:"timestamp"`
	SessionID    string                 `bson:"session_id,omitempty" json:"session_id"`
}

// Inventory model - MongoDB (flexible inventory tracking)
type Inventory struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	ProductID        primitive.ObjectID     `bson:"product_id" json:"product_id"`
	RestaurantID     string                 `bson:"restaurant_id" json:"restaurant_id"`
	Quantity         int                    `bson:"quantity" json:"quantity"`
	ReservedQuantity int                    `bson:"reserved_quantity" json:"reserved_quantity"`
	MinStockLevel    int                    `bson:"min_stock_level" json:"min_stock_level"`
	MaxStockLevel    int                    `bson:"max_stock_level" json:"max_stock_level"`
	LastRestocked    time.Time              `bson:"last_restocked" json:"last_restocked"`
	UpdatedAt        time.Time              `bson:"updated_at" json:"updated_at"`
	StockHistory     []StockTransaction     `bson:"stock_history,omitempty" json:"stock_history"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty" json:"metadata"`
}

type StockTransaction struct {
	Type      string    `bson:"type" json:"type"` // addition, deduction, reserved, released
	Quantity  int       `bson:"quantity" json:"quantity"`
	Reason    string    `bson:"reason" json:"reason"`
	Reference string    `bson:"reference,omitempty" json:"reference"` // order_id, manual, etc
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

// MenuSection model - MongoDB (for organizing menu items)
type MenuSection struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID    string             `bson:"restaurant_id" json:"restaurant_id"`
	Name            string             `bson:"name" json:"name"`
	Description     string             `bson:"description,omitempty" json:"description"`
	SortOrder       int                `bson:"sort_order" json:"sort_order"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	TimeRestriction *TimeRestriction   `bson:"time_restriction,omitempty" json:"time_restriction"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

type TimeRestriction struct {
	StartTime string   `bson:"start_time" json:"start_time"` // HH:MM format
	EndTime   string   `bson:"end_time" json:"end_time"`     // HH:MM format
	Days      []string `bson:"days" json:"days"`             // monday, tuesday, etc
}

// RestaurantAnalytics model - MongoDB (for restaurant insights)
type RestaurantAnalytics struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	RestaurantID  string                 `bson:"restaurant_id" json:"restaurant_id"`
	Date          string                 `bson:"date" json:"date"` // YYYY-MM-DD format
	TotalOrders   int                    `bson:"total_orders" json:"total_orders"`
	TotalRevenue  float64                `bson:"total_revenue" json:"total_revenue"`
	PopularItems  []PopularItem          `bson:"popular_items" json:"popular_items"`
	PeakHours     map[string]int         `bson:"peak_hours" json:"peak_hours"`
	CustomerStats CustomerStats          `bson:"customer_stats" json:"customer_stats"`
	Metadata      map[string]interface{} `bson:"metadata,omitempty" json:"metadata"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
}

type PopularItem struct {
	ProductID  primitive.ObjectID `bson:"product_id" json:"product_id"`
	Name       string             `bson:"name" json:"name"`
	OrderCount int                `bson:"order_count" json:"order_count"`
	Revenue    float64            `bson:"revenue" json:"revenue"`
}

type CustomerStats struct {
	NewCustomers    int `bson:"new_customers" json:"new_customers"`
	ReturnCustomers int `bson:"return_customers" json:"return_customers"`
	TotalCustomers  int `bson:"total_customers" json:"total_customers"`
}
