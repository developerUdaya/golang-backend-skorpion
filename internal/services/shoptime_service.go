package services

import (
	"context"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/cache"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ShopTimeService struct {
	restaurantRepo       repositories.RestaurantRepository
	timeRangeProductRepo repositories.TimeRangeProductRepository
	productRepo          repositories.ProductRepository
	cache                *cache.RedisCache
}

func NewShopTimeService(
	restaurantRepo repositories.RestaurantRepository,
	timeRangeProductRepo repositories.TimeRangeProductRepository,
	productRepo repositories.ProductRepository,
	cache *cache.RedisCache,
) *ShopTimeService {
	return &ShopTimeService{
		restaurantRepo:       restaurantRepo,
		timeRangeProductRepo: timeRangeProductRepo,
		productRepo:          productRepo,
		cache:                cache,
	}
}

// Request/Response types for shop timing
type UpdateShopTimingRequest struct {
	OpeningHours  map[string]DayTiming `json:"opening_hours" binding:"required"`
	AutoOpenClose bool                 `json:"auto_open_close"`
}

type DayTiming struct {
	IsOpen    bool   `json:"is_open"`
	OpenTime  string `json:"open_time"`  // HH:MM format
	CloseTime string `json:"close_time"` // HH:MM format
}

type ShopStatusRequest struct {
	IsOpen bool `json:"is_open" binding:"required"`
}

// Request/Response types for time-based products
type CreateTimeGroupRequest struct {
	RestaurantID string `json:"restaurant_id" binding:"required"`
	GroupName    string `json:"group_name" binding:"required"`
	StartTime    string `json:"start_time" binding:"required"` // HH:MM format
	EndTime      string `json:"end_time" binding:"required"`   // HH:MM format
}

type UpdateTimeGroupRequest struct {
	GroupName string `json:"group_name"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	IsActive  *bool  `json:"is_active"`
}

type AddProductToTimeGroupRequest struct {
	ProductID string `json:"product_id" binding:"required"`
}

type TimeBasedProductResponse struct {
	Products         []models.Product                `json:"products"`
	TimeGroups       []models.TimeRangeProductsGroup `json:"time_groups"`
	CurrentTime      string                          `json:"current_time"`
	IsRestaurantOpen bool                            `json:"is_restaurant_open"`
}

// Shop timing management methods
func (s *ShopTimeService) UpdateShopTiming(ctx context.Context, restaurantID string, req *UpdateShopTimingRequest) (*models.Restaurant, error) {
	restaurantUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	// Get restaurant
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant not found: %v", err)
	}

	// Convert opening hours to JSONB
	openingHoursJSON := make(models.JSONB)
	for day, timing := range req.OpeningHours {
		openingHoursJSON[day] = map[string]interface{}{
			"is_open":    timing.IsOpen,
			"open_time":  timing.OpenTime,
			"close_time": timing.CloseTime,
		}
	}

	// Update restaurant
	restaurant.OpeningHours = openingHoursJSON
	restaurant.AutoOpenClose = req.AutoOpenClose

	if err := s.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to update restaurant: %v", err)
	}

	// Clear cache
	s.cache.DeleteWithPrefix(ctx, "restaurant", restaurantID)

	return restaurant, nil
}

func (s *ShopTimeService) GetShopTiming(ctx context.Context, restaurantID string) (*models.Restaurant, error) {
	restaurantUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	// Try cache first
	var restaurant models.Restaurant
	cacheKey := fmt.Sprintf("restaurant:%s:timing", restaurantID)
	if err := s.cache.Get(ctx, cacheKey, &restaurant); err == nil {
		return &restaurant, nil
	}

	// Get from database
	restaurantPtr, err := s.restaurantRepo.GetByID(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant not found: %v", err)
	}

	// Cache for 5 minutes
	s.cache.Set(ctx, cacheKey, restaurantPtr, 5*time.Minute)

	return restaurantPtr, nil
}

func (s *ShopTimeService) UpdateShopStatus(ctx context.Context, restaurantID string, req *ShopStatusRequest) (*models.Restaurant, error) {
	restaurantUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	// Get restaurant
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant not found: %v", err)
	}

	// Update status
	restaurant.IsOpen = req.IsOpen

	if err := s.restaurantRepo.Update(ctx, restaurant); err != nil {
		return nil, fmt.Errorf("failed to update restaurant status: %v", err)
	}

	// Clear cache
	s.cache.DeleteWithPrefix(ctx, "restaurant", restaurantID)

	return restaurant, nil
}

func (s *ShopTimeService) AutoUpdateShopStatus(ctx context.Context) error {
	// This method should be called by a cron job
	// Get all restaurants with auto_open_close enabled
	// For now, we'll implement a basic version

	currentTime := time.Now()
	currentTimeStr := currentTime.Format("15:04")
	currentDay := currentTime.Weekday().String()

	// In a real implementation, you'd get all restaurants and update their status
	// based on their opening hours
	fmt.Printf("Auto updating shop status for time: %s, day: %s\n", currentTimeStr, currentDay)

	return nil
}

// Time-based product management methods
func (s *ShopTimeService) CreateTimeGroup(ctx context.Context, req *CreateTimeGroupRequest) (*models.TimeRangeProductsGroup, error) {
	group := &models.TimeRangeProductsGroup{
		RestaurantID: req.RestaurantID,
		GroupName:    req.GroupName,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		IsActive:     true,
	}

	if err := s.timeRangeProductRepo.CreateTimeGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create time group: %v", err)
	}

	return group, nil
}

func (s *ShopTimeService) GetTimeGroups(ctx context.Context, restaurantID string) ([]models.TimeRangeProductsGroup, error) {
	return s.timeRangeProductRepo.GetTimeGroupsByRestaurant(ctx, restaurantID)
}

func (s *ShopTimeService) UpdateTimeGroup(ctx context.Context, groupID string, req *UpdateTimeGroupRequest) (*models.TimeRangeProductsGroup, error) {
	groupObjectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %v", err)
	}

	group, err := s.timeRangeProductRepo.GetTimeGroupByID(ctx, groupObjectID)
	if err != nil {
		return nil, fmt.Errorf("time group not found: %v", err)
	}

	// Update fields
	if req.GroupName != "" {
		group.GroupName = req.GroupName
	}
	if req.StartTime != "" {
		group.StartTime = req.StartTime
	}
	if req.EndTime != "" {
		group.EndTime = req.EndTime
	}
	if req.IsActive != nil {
		group.IsActive = *req.IsActive
	}

	if err := s.timeRangeProductRepo.UpdateTimeGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to update time group: %v", err)
	}

	return group, nil
}

func (s *ShopTimeService) DeleteTimeGroup(ctx context.Context, groupID string) error {
	groupObjectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}

	return s.timeRangeProductRepo.DeleteTimeGroup(ctx, groupObjectID)
}

func (s *ShopTimeService) AddProductToTimeGroup(ctx context.Context, groupID string, req *AddProductToTimeGroupRequest) error {
	groupObjectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}

	productObjectID, err := primitive.ObjectIDFromHex(req.ProductID)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	item := &models.TimeRangeProductsGroupItem{
		ProductID: productObjectID,
		GroupID:   groupObjectID,
	}

	return s.timeRangeProductRepo.AddProductToTimeGroup(ctx, item)
}

func (s *ShopTimeService) RemoveProductFromTimeGroup(ctx context.Context, groupID, productID string) error {
	groupObjectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}

	productObjectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	return s.timeRangeProductRepo.RemoveProductFromTimeGroup(ctx, groupObjectID, productObjectID)
}

func (s *ShopTimeService) GetProductsByTime(ctx context.Context, restaurantID string, targetTime *string) (*TimeBasedProductResponse, error) {
	currentTime := time.Now()
	timeStr := currentTime.Format("15:04")

	if targetTime != nil {
		timeStr = *targetTime
	}

	// Get time-based products
	activeProductIDs, err := s.timeRangeProductRepo.GetActiveProductsByTime(ctx, restaurantID, timeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get active products: %v", err)
	}

	// Get product details for active products
	var products []models.Product
	for _, productID := range activeProductIDs {
		product, err := s.productRepo.GetByID(ctx, productID)
		if err != nil {
			continue // Skip if product not found
		}
		products = append(products, *product)
	}

	// Get time groups
	timeGroups, err := s.timeRangeProductRepo.GetTimeGroupsByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time groups: %v", err)
	}

	// Check if restaurant is open
	restaurant, err := s.GetShopTiming(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant timing: %v", err)
	}

	isOpen := s.isRestaurantOpenAtTime(restaurant, currentTime)

	return &TimeBasedProductResponse{
		Products:         products,
		TimeGroups:       timeGroups,
		CurrentTime:      timeStr,
		IsRestaurantOpen: isOpen,
	}, nil
}

func (s *ShopTimeService) isRestaurantOpenAtTime(restaurant *models.Restaurant, checkTime time.Time) bool {
	if !restaurant.AutoOpenClose {
		return restaurant.IsOpen
	}

	// Check opening hours
	currentDay := checkTime.Weekday().String()
	currentTimeStr := checkTime.Format("15:04")

	if restaurant.OpeningHours == nil {
		return restaurant.IsOpen
	}

	dayTiming, exists := restaurant.OpeningHours[currentDay]
	if !exists {
		return false
	}

	dayTimingMap, ok := dayTiming.(map[string]interface{})
	if !ok {
		return false
	}

	isOpen, _ := dayTimingMap["is_open"].(bool)
	if !isOpen {
		return false
	}

	openTime, _ := dayTimingMap["open_time"].(string)
	closeTime, _ := dayTimingMap["close_time"].(string)

	if openTime == "" || closeTime == "" {
		return restaurant.IsOpen
	}

	return currentTimeStr >= openTime && currentTimeStr <= closeTime
}
