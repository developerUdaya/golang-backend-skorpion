package services

import (
	"context"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/cache"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TimeBasedProductService struct {
	productRepo          repositories.ProductRepository
	timeRangeProductRepo repositories.TimeRangeProductRepository
	restaurantRepo       repositories.RestaurantRepository
	cache                *cache.RedisCache
}

func NewTimeBasedProductService(
	productRepo repositories.ProductRepository,
	timeRangeProductRepo repositories.TimeRangeProductRepository,
	restaurantRepo repositories.RestaurantRepository,
	cache *cache.RedisCache,
) *TimeBasedProductService {
	return &TimeBasedProductService{
		productRepo:          productRepo,
		timeRangeProductRepo: timeRangeProductRepo,
		restaurantRepo:       restaurantRepo,
		cache:                cache,
	}
}

// Advanced product fetching requests
type GetProductsByTimeRequest struct {
	RestaurantID string       `json:"restaurant_id" binding:"required"`
	DateTime     *string      `json:"date_time,omitempty"` // ISO format: 2025-01-02T14:30:00Z
	Time         *string      `json:"time,omitempty"`      // HH:MM format: 14:30
	Date         *string      `json:"date,omitempty"`      // YYYY-MM-DD format: 2025-01-02
	Category     *string      `json:"category,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	PriceRange   *PriceFilter `json:"price_range,omitempty"`
	Availability *bool        `json:"availability,omitempty"`
	Page         int          `json:"page"`
	Limit        int          `json:"limit"`
}

type PriceFilter struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

type ProductTimeInfo struct {
	ProductID     string                          `json:"product_id"`
	Product       *models.Product                 `json:"product"`
	TimeGroups    []models.TimeRangeProductsGroup `json:"time_groups"`
	IsAvailable   bool                            `json:"is_available"`
	NextAvailable *string                         `json:"next_available,omitempty"` // Next time this product will be available
	Reason        string                          `json:"reason,omitempty"`         // Why product is not available
}

type TimeBasedProductsResponse struct {
	Products         []ProductTimeInfo               `json:"products"`
	TotalCount       int                             `json:"total_count"`
	Page             int                             `json:"page"`
	Limit            int                             `json:"limit"`
	CurrentTime      string                          `json:"current_time"`
	RequestedTime    string                          `json:"requested_time"`
	TimeGroups       []models.TimeRangeProductsGroup `json:"time_groups"`
	RestaurantStatus RestaurantTimeStatus            `json:"restaurant_status"`
}

type RestaurantTimeStatus struct {
	IsOpen               bool       `json:"is_open"`
	OpeningTime          *string    `json:"opening_time,omitempty"`
	ClosingTime          *string    `json:"closing_time,omitempty"`
	NextOpenTime         *string    `json:"next_open_time,omitempty"`
	AutoOpenCloseEnabled bool       `json:"auto_open_close_enabled"`
	LastStatusUpdate     *time.Time `json:"last_status_update,omitempty"`
	TimeZone             string     `json:"timezone"`
}

// GetProductsByTimeAdvanced - Enhanced product fetching with time-based filtering
func (s *TimeBasedProductService) GetProductsByTimeAdvanced(ctx context.Context, req *GetProductsByTimeRequest) (*TimeBasedProductsResponse, error) {
	// Parse and validate time parameters
	targetTime, err := s.parseTimeParameters(req)
	if err != nil {
		return nil, fmt.Errorf("invalid time parameters: %v", err)
	}

	// Get restaurant status and timezone info
	restaurantStatus, err := s.getRestaurantTimeStatus(ctx, req.RestaurantID, targetTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant status: %v", err)
	}

	// Get all time groups for the restaurant
	timeGroups, err := s.timeRangeProductRepo.GetTimeGroupsByRestaurant(ctx, req.RestaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time groups: %v", err)
	}

	// Get products available at the specified time
	timeStr := targetTime.Format("15:04")
	activeProductIDs, err := s.timeRangeProductRepo.GetActiveProductsByTime(ctx, req.RestaurantID, timeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get active products: %v", err)
	}

	// Get all restaurant products for filtering
	allProducts, err := s.productRepo.GetByRestaurantID(ctx, req.RestaurantID, 1000, 0) // Get a large number
	if err != nil {
		return nil, fmt.Errorf("failed to get restaurant products: %v", err)
	}

	// Apply filters and build response
	filteredProducts := s.applyFilters(allProducts, activeProductIDs, req)
	productInfos := s.buildProductTimeInfos(ctx, filteredProducts, timeGroups, targetTime, restaurantStatus.IsOpen)

	// Apply pagination
	startIdx := (req.Page - 1) * req.Limit
	endIdx := startIdx + req.Limit
	if endIdx > len(productInfos) {
		endIdx = len(productInfos)
	}
	if startIdx > len(productInfos) {
		startIdx = len(productInfos)
	}

	paginatedProducts := productInfos[startIdx:endIdx]

	return &TimeBasedProductsResponse{
		Products:         paginatedProducts,
		TotalCount:       len(productInfos),
		Page:             req.Page,
		Limit:            req.Limit,
		CurrentTime:      time.Now().Format("15:04"),
		RequestedTime:    timeStr,
		TimeGroups:       timeGroups,
		RestaurantStatus: *restaurantStatus,
	}, nil
}

// parseTimeParameters parses the various time formats from the request
func (s *TimeBasedProductService) parseTimeParameters(req *GetProductsByTimeRequest) (time.Time, error) {
	now := time.Now()

	// If DateTime is provided in ISO format
	if req.DateTime != nil {
		parsedTime, err := time.Parse(time.RFC3339, *req.DateTime)
		if err != nil {
			return now, fmt.Errorf("invalid datetime format, expected RFC3339: %v", err)
		}
		return parsedTime, nil
	}

	// If Date and Time are provided separately
	if req.Date != nil && req.Time != nil {
		dateTimeStr := fmt.Sprintf("%sT%s:00", *req.Date, *req.Time)
		parsedTime, err := time.Parse("2006-01-02T15:04:05", dateTimeStr)
		if err != nil {
			return now, fmt.Errorf("invalid date/time format: %v", err)
		}
		return parsedTime, nil
	}

	// If only Time is provided, use today's date
	if req.Time != nil {
		todayStr := now.Format("2006-01-02")
		dateTimeStr := fmt.Sprintf("%sT%s:00", todayStr, *req.Time)
		parsedTime, err := time.Parse("2006-01-02T15:04:05", dateTimeStr)
		if err != nil {
			return now, fmt.Errorf("invalid time format: %v", err)
		}
		return parsedTime, nil
	}

	// If only Date is provided, use current time
	if req.Date != nil {
		currentTimeStr := now.Format("15:04:05")
		dateTimeStr := fmt.Sprintf("%sT%s", *req.Date, currentTimeStr)
		parsedTime, err := time.Parse("2006-01-02T15:04:05", dateTimeStr)
		if err != nil {
			return now, fmt.Errorf("invalid date format: %v", err)
		}
		return parsedTime, nil
	}

	// Default to current time
	return now, nil
}

// getRestaurantTimeStatus gets comprehensive restaurant timing status
func (s *TimeBasedProductService) getRestaurantTimeStatus(ctx context.Context, restaurantID string, checkTime time.Time) (*RestaurantTimeStatus, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("restaurant:%s:time_status", restaurantID)
	var cached RestaurantTimeStatus
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Get from database
	restaurant, err := s.getRestaurantByID(ctx, restaurantID)
	if err != nil {
		return nil, err
	}

	status := &RestaurantTimeStatus{
		IsOpen:               restaurant.IsOpen,
		AutoOpenCloseEnabled: restaurant.AutoOpenClose,
		LastStatusUpdate:     restaurant.LastStatusUpdate,
		TimeZone:             restaurant.TimeZone,
	}

	// Calculate opening/closing times if auto open/close is enabled
	if restaurant.AutoOpenClose && restaurant.OpeningHours != nil {
		s.calculateOperatingHours(status, restaurant, checkTime)
	}

	// Cache for 1 minute
	s.cache.Set(ctx, cacheKey, status, time.Minute)

	return status, nil
}

// calculateOperatingHours calculates next open/close times
func (s *TimeBasedProductService) calculateOperatingHours(status *RestaurantTimeStatus, restaurant *models.Restaurant, checkTime time.Time) {
	dayOfWeek := strings.ToLower(checkTime.Weekday().String())

	if dayTiming, exists := restaurant.OpeningHours[dayOfWeek]; exists {
		if dayMap, ok := dayTiming.(map[string]interface{}); ok {
			if isOpen, _ := dayMap["is_open"].(bool); isOpen {
				if openTime, ok := dayMap["open_time"].(string); ok {
					status.OpeningTime = &openTime
				}
				if closeTime, ok := dayMap["close_time"].(string); ok {
					status.ClosingTime = &closeTime
				}
			}
		}
	}

	// Calculate next opening time if restaurant is closed
	if !status.IsOpen {
		nextOpen := s.findNextOpenTime(restaurant, checkTime)
		if nextOpen != nil {
			nextOpenStr := nextOpen.Format("2006-01-02T15:04:05Z07:00")
			status.NextOpenTime = &nextOpenStr
		}
	}
}

// findNextOpenTime finds the next time the restaurant will be open
func (s *TimeBasedProductService) findNextOpenTime(restaurant *models.Restaurant, currentTime time.Time) *time.Time {
	// Look for next 7 days
	for i := 0; i < 7; i++ {
		checkDate := currentTime.AddDate(0, 0, i)
		dayOfWeek := strings.ToLower(checkDate.Weekday().String())

		if dayTiming, exists := restaurant.OpeningHours[dayOfWeek]; exists {
			if dayMap, ok := dayTiming.(map[string]interface{}); ok {
				if isOpen, _ := dayMap["is_open"].(bool); isOpen {
					if openTimeStr, ok := dayMap["open_time"].(string); ok {
						// Parse the open time for this day
						openDateTime, err := time.Parse("2006-01-02T15:04",
							fmt.Sprintf("%s%s", checkDate.Format("2006-01-02T"), openTimeStr))
						if err == nil && (i > 0 || openDateTime.After(currentTime)) {
							return &openDateTime
						}
					}
				}
			}
		}
	}
	return nil
}

// applyFilters applies various filters to the products
func (s *TimeBasedProductService) applyFilters(allProducts []models.Product, activeProductIDs []primitive.ObjectID, req *GetProductsByTimeRequest) []models.Product {
	activeProductMap := make(map[primitive.ObjectID]bool)
	for _, id := range activeProductIDs {
		activeProductMap[id] = true
	}

	var filtered []models.Product
	for _, product := range allProducts {
		// Check if product is in active time range (if time filtering is used)
		if len(activeProductIDs) > 0 && !activeProductMap[product.ID] {
			continue
		}

		// Apply category filter (note: this would need category lookup by ID)
		if req.Category != nil {
			// In a real implementation, you would look up category name by CategoryID
			// For now, we'll skip this filter or implement category lookup
		}

		// Apply tag filters
		if len(req.Tags) > 0 && !s.hasAnyTag(product.Tags, req.Tags) {
			continue
		}

		// Apply price range filter
		if req.PriceRange != nil {
			if req.PriceRange.Min != nil && product.Price < *req.PriceRange.Min {
				continue
			}
			if req.PriceRange.Max != nil && product.Price > *req.PriceRange.Max {
				continue
			}
		}

		// Apply availability filter
		if req.Availability != nil && product.IsAvailable != *req.Availability {
			continue
		}

		filtered = append(filtered, product)
	}

	return filtered
}

// hasAnyTag checks if product has any of the requested tags
func (s *TimeBasedProductService) hasAnyTag(productTags, requestedTags []string) bool {
	for _, requestedTag := range requestedTags {
		for _, productTag := range productTags {
			if strings.EqualFold(productTag, requestedTag) {
				return true
			}
		}
	}
	return false
}

// buildProductTimeInfos builds detailed product information with time context
func (s *TimeBasedProductService) buildProductTimeInfos(ctx context.Context, products []models.Product, timeGroups []models.TimeRangeProductsGroup, targetTime time.Time, restaurantIsOpen bool) []ProductTimeInfo {
	var result []ProductTimeInfo

	for _, product := range products {
		info := ProductTimeInfo{
			ProductID: product.ID.Hex(),
			Product:   &product,
		}

		// Find which time groups this product belongs to
		for _, group := range timeGroups {
			if s.productBelongsToGroup(ctx, product.ID, group.ID) {
				info.TimeGroups = append(info.TimeGroups, group)
			}
		}

		// Determine availability
		info.IsAvailable = s.isProductAvailable(product, info.TimeGroups, targetTime, restaurantIsOpen)

		if !info.IsAvailable {
			info.Reason = s.getUnavailabilityReason(product, info.TimeGroups, targetTime, restaurantIsOpen)
			info.NextAvailable = s.findNextAvailableTime(info.TimeGroups, targetTime)
		}

		result = append(result, info)
	}

	return result
}

// Helper methods
func (s *TimeBasedProductService) getRestaurantByID(ctx context.Context, restaurantID string) (*models.Restaurant, error) {
	// Parse UUID
	uuid, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	return s.restaurantRepo.GetByID(ctx, uuid)
}

func (s *TimeBasedProductService) productBelongsToGroup(ctx context.Context, productID, groupID primitive.ObjectID) bool {
	items, err := s.timeRangeProductRepo.GetProductsByTimeGroup(ctx, groupID)
	if err != nil {
		return false
	}

	for _, item := range items {
		if item.ProductID == productID {
			return true
		}
	}
	return false
}

func (s *TimeBasedProductService) isProductAvailable(product models.Product, timeGroups []models.TimeRangeProductsGroup, targetTime time.Time, restaurantIsOpen bool) bool {
	if !restaurantIsOpen {
		return false
	}

	if !product.IsAvailable {
		return false
	}

	// If product has no time restrictions, it's available
	if len(timeGroups) == 0 {
		return true
	}

	// Check if current time falls within any of the product's time groups
	targetTimeStr := targetTime.Format("15:04")
	for _, group := range timeGroups {
		if group.IsActive && s.isTimeInRange(targetTimeStr, group.StartTime, group.EndTime) {
			return true
		}
	}

	return false
}

func (s *TimeBasedProductService) isTimeInRange(checkTime, startTime, endTime string) bool {
	// Handle overnight ranges (e.g., 22:00 - 06:00)
	if endTime < startTime {
		return checkTime >= startTime || checkTime <= endTime
	}
	return checkTime >= startTime && checkTime <= endTime
}

func (s *TimeBasedProductService) getUnavailabilityReason(product models.Product, timeGroups []models.TimeRangeProductsGroup, targetTime time.Time, restaurantIsOpen bool) string {
	if !restaurantIsOpen {
		return "Restaurant is closed"
	}

	if !product.IsAvailable {
		return "Product is currently unavailable"
	}

	if len(timeGroups) > 0 {
		return "Product not available at this time"
	}

	return "Unknown reason"
}

func (s *TimeBasedProductService) findNextAvailableTime(timeGroups []models.TimeRangeProductsGroup, currentTime time.Time) *string {
	if len(timeGroups) == 0 {
		return nil
	}

	// Find the next time group that will be active
	// This is a simplified implementation
	for _, group := range timeGroups {
		if group.IsActive {
			// Return the start time of the next active group
			// In a real implementation, you'd calculate this more precisely
			return &group.StartTime
		}
	}

	return nil
}
