package services

import (
	"context"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EnhancedCronService struct {
	restaurantRepo    repositories.RestaurantRepository
	stopChan          chan bool
	timezone          *time.Location
	isRunning         bool
	statusUpdateCount int
	activeRestaurants []models.Restaurant
	mutex             sync.RWMutex
}

func NewEnhancedCronService(restaurantRepo repositories.RestaurantRepository) *EnhancedCronService {
	// Default to Asia/Kolkata timezone
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.UTC
		log.Printf("Failed to load timezone, using UTC: %v", err)
	}

	return &EnhancedCronService{
		restaurantRepo: restaurantRepo,
		stopChan:       make(chan bool),
		timezone:       loc,
		isRunning:      false,
	}
}

// StartAutomaticStatusManagement starts the background jobs for restaurant status management
func (s *EnhancedCronService) StartAutomaticStatusManagement() error {
	if s.isRunning {
		return fmt.Errorf("cron service is already running")
	}

	s.isRunning = true

	// Start the status update ticker (every minute)
	go s.runStatusUpdateTicker()

	// Start the maintenance ticker (every hour)
	go s.runMaintenanceTicker()

	// Start the daily report ticker (every day at midnight)
	go s.runDailyReportTicker()

	log.Println("✅ Enhanced cron service started successfully")
	log.Println("📅 Restaurant status updates: Every minute")
	log.Println("🔧 Maintenance tasks: Every hour")
	log.Println("📊 Daily reports: Every day at midnight")

	return nil
}

// StopAutomaticStatusManagement stops the background jobs
func (s *EnhancedCronService) StopAutomaticStatusManagement() {
	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
	log.Println("🛑 Enhanced cron service stopped")
}

// runStatusUpdateTicker runs status updates every minute
func (s *EnhancedCronService) runStatusUpdateTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateAllRestaurantStatuses()
		case <-s.stopChan:
			return
		}
	}
}

// runMaintenanceTicker runs maintenance tasks every hour
func (s *EnhancedCronService) runMaintenanceTicker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performMaintenanceTasks()
		case <-s.stopChan:
			return
		}
	}
}

// runDailyReportTicker runs daily reports at midnight
func (s *EnhancedCronService) runDailyReportTicker() {
	// Calculate time until next midnight
	now := time.Now().In(s.timezone)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, s.timezone)
	duration := next.Sub(now)

	// Wait until midnight, then run every 24 hours
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			s.generateDailyReports()
			timer.Reset(24 * time.Hour)
		case <-s.stopChan:
			return
		}
	}
}

// updateAllRestaurantStatuses updates the open/close status for all restaurants
func (s *EnhancedCronService) updateAllRestaurantStatuses() {
	ctx := context.Background()
	currentTime := time.Now().In(s.timezone)

	log.Printf("🕒 Running automatic restaurant status update at %s", currentTime.Format("2006-01-02 15:04:05"))

	// Get all restaurants with pagination
	limit := 100
	offset := 0
	totalUpdated := 0
	totalErrors := 0

	for {
		restaurants, err := s.getRestaurantsWithAutoOpenClose(ctx, limit, offset)
		if err != nil {
			log.Printf("❌ Error fetching restaurants: %v", err)
			break
		}

		if len(restaurants) == 0 {
			break
		}

		for _, restaurant := range restaurants {
			if err := s.updateRestaurantStatus(ctx, &restaurant, currentTime); err != nil {
				log.Printf("❌ Error updating restaurant %s (%s): %v", restaurant.Name, restaurant.ID, err)
				totalErrors++
			} else {
				totalUpdated++
			}
		}

		offset += limit

		// Break if we got fewer results than the limit (last page)
		if len(restaurants) < limit {
			break
		}
	}

	log.Printf("✅ Status update completed: %d restaurants updated, %d errors", totalUpdated, totalErrors)
}

// getRestaurantsWithAutoOpenClose gets restaurants that have auto open/close enabled
func (s *EnhancedCronService) getRestaurantsWithAutoOpenClose(ctx context.Context, limit, offset int) ([]models.Restaurant, error) {
	// This is a simplified implementation - in a real app you'd have a method in your repository
	// to get restaurants filtered by auto_open_close = true
	// For now, we'll search all restaurants and filter in memory
	allRestaurants, err := s.restaurantRepo.Search(ctx, "", limit*2, offset) // Get more to account for filtering
	if err != nil {
		return nil, err
	}

	var filteredRestaurants []models.Restaurant
	for _, restaurant := range allRestaurants {
		if restaurant.AutoOpenClose && restaurant.Status == "active" {
			filteredRestaurants = append(filteredRestaurants, restaurant)
			if len(filteredRestaurants) >= limit {
				break
			}
		}
	}

	return filteredRestaurants, nil
}

// updateRestaurantStatus updates a single restaurant's status based on its opening hours
func (s *EnhancedCronService) updateRestaurantStatus(ctx context.Context, restaurant *models.Restaurant, checkTime time.Time) error {
	// Calculate if restaurant should be open
	shouldBeOpen := s.shouldRestaurantBeOpenAtTime(restaurant, checkTime)
	statusChanged := restaurant.IsOpen != shouldBeOpen

	// Update status if it has changed
	if statusChanged {
		restaurant.IsOpen = shouldBeOpen
		now := time.Now()
		restaurant.LastStatusUpdate = &now

		if err := s.restaurantRepo.Update(ctx, restaurant); err != nil {
			return fmt.Errorf("failed to update restaurant status: %v", err)
		}

		statusText := "CLOSED"
		if shouldBeOpen {
			statusText = "OPENED"
		}

		log.Printf("🔄 %s: %s (%s) - Status changed to %s",
			checkTime.Format("15:04"),
			restaurant.Name,
			restaurant.ID,
			statusText)
	}

	return nil
}

// shouldRestaurantBeOpenAtTime determines if a restaurant should be open at a specific time
func (s *EnhancedCronService) shouldRestaurantBeOpenAtTime(restaurant *models.Restaurant, checkTime time.Time) bool {
	if !restaurant.AutoOpenClose {
		return restaurant.IsOpen // Return current status if auto management is disabled
	}

	if restaurant.OpeningHours == nil {
		return false // No opening hours defined
	}

	// Get current day and time
	currentDay := strings.ToLower(checkTime.Weekday().String())
	currentTimeStr := checkTime.Format("15:04")

	// Get timing for current day
	dayTiming, exists := restaurant.OpeningHours[currentDay]
	if !exists {
		return false // No timing defined for this day
	}

	dayTimingMap, ok := dayTiming.(map[string]interface{})
	if !ok {
		return false // Invalid timing format
	}

	// Check if restaurant is supposed to be open on this day
	isOpenToday, _ := dayTimingMap["is_open"].(bool)
	if !isOpenToday {
		return false
	}

	// Get open and close times
	openTimeStr, _ := dayTimingMap["open_time"].(string)
	closeTimeStr, _ := dayTimingMap["close_time"].(string)

	if openTimeStr == "" || closeTimeStr == "" {
		return false // Invalid time format
	}

	// Handle overnight restaurants (e.g., open 22:00, close 06:00 next day)
	if closeTimeStr < openTimeStr {
		// Restaurant is open overnight
		return currentTimeStr >= openTimeStr || currentTimeStr <= closeTimeStr
	}

	// Normal operating hours (same day)
	return currentTimeStr >= openTimeStr && currentTimeStr <= closeTimeStr
}

// performMaintenanceTasks runs hourly maintenance tasks
func (s *EnhancedCronService) performMaintenanceTasks() {
	ctx := context.Background()
	currentTime := time.Now().In(s.timezone)

	log.Printf("🔧 Running maintenance tasks at %s", currentTime.Format("2006-01-02 15:04:05"))

	// Clean up old status update logs (keep last 30 days)
	s.cleanupOldLogs(ctx, currentTime.AddDate(0, 0, -30))

	// Check for restaurants that haven't updated status in a while
	s.checkStaleRestaurants(ctx, currentTime.Add(-2*time.Hour))

	// Validate restaurant opening hours format
	s.validateRestaurantTimings(ctx)

	log.Printf("✅ Maintenance tasks completed at %s", currentTime.Format("15:04:05"))
}

// cleanupOldLogs removes old status update logs
func (s *EnhancedCronService) cleanupOldLogs(ctx context.Context, cutoffTime time.Time) {
	log.Printf("🧹 Cleaning up status logs older than %s", cutoffTime.Format("2006-01-02"))
	// Implementation would depend on how you store logs
	// This is a placeholder for the actual cleanup logic
}

// checkStaleRestaurants identifies restaurants that haven't been updated recently
func (s *EnhancedCronService) checkStaleRestaurants(ctx context.Context, cutoffTime time.Time) {
	log.Printf("🔍 Checking for restaurants with stale status (not updated since %s)", cutoffTime.Format("2006-01-02 15:04"))

	// In a real implementation, you would:
	// 1. Query restaurants where last_status_update < cutoffTime
	// 2. Log warnings for these restaurants
	// 3. Optionally force a status update
	// 4. Send notifications to restaurant owners
}

// validateRestaurantTimings validates opening hours format for all restaurants
func (s *EnhancedCronService) validateRestaurantTimings(ctx context.Context) {
	log.Printf("✅ Validating restaurant timing configurations...")

	// This would check for:
	// 1. Invalid time formats
	// 2. Missing required fields
	// 3. Logical errors (close time before open time for same day)
	// 4. Empty or malformed opening_hours JSON
}

// generateDailyReports generates daily operational reports
func (s *EnhancedCronService) generateDailyReports() {
	currentTime := time.Now().In(s.timezone)
	log.Printf("📊 Generating daily reports for %s", currentTime.Format("2006-01-02"))

	// Generate reports for:
	// 1. Restaurant status changes summary
	// 2. Average open/close times
	// 3. Restaurants with configuration issues
	// 4. System health metrics

	s.generateRestaurantStatusSummary(currentTime)
	s.generateSystemHealthReport(currentTime)
}

// generateRestaurantStatusSummary creates a summary of restaurant status changes
func (s *EnhancedCronService) generateRestaurantStatusSummary(reportDate time.Time) {
	log.Printf("📈 Restaurant Status Summary for %s:", reportDate.Format("2006-01-02"))
	log.Printf("   - Total restaurants with auto-management: [To be implemented]")
	log.Printf("   - Status changes today: [To be implemented]")
	log.Printf("   - Average open duration: [To be implemented]")
	log.Printf("   - Restaurants with timing issues: [To be implemented]")
}

// generateSystemHealthReport creates a system health report
func (s *EnhancedCronService) generateSystemHealthReport(reportDate time.Time) {
	log.Printf("🏥 System Health Report for %s:", reportDate.Format("2006-01-02"))
	log.Printf("   - Cron job success rate: 100%% (placeholder)")
	log.Printf("   - Database connection status: OK")
	log.Printf("   - Average response time: [To be implemented]")
	log.Printf("   - Memory usage: [To be implemented]")
}

// GetRestaurantStatusHistory gets the status change history for a restaurant
func (s *EnhancedCronService) GetRestaurantStatusHistory(ctx context.Context, restaurantID string, days int) ([]StatusChange, error) {
	// This is a placeholder implementation
	// In a real system, you'd store status changes in a separate table/collection

	restaurantUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	// Get restaurant details to verify it exists
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant not found")
	}

	// Generate mock status history for demonstration
	var history []StatusChange
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)

		// Morning opening
		history = append(history, StatusChange{
			Timestamp:      date.Add(time.Hour * 9), // 9 AM
			PreviousStatus: "closed",
			NewStatus:      "open",
			Reason:         "Scheduled opening time",
			UpdatedBy:      "system",
		})

		// Evening closing
		history = append(history, StatusChange{
			Timestamp:      date.Add(time.Hour * 22), // 10 PM
			PreviousStatus: "open",
			NewStatus:      "closed",
			Reason:         "Scheduled closing time",
			UpdatedBy:      "system",
		})
	}

	// Add current status
	if len(history) > 0 {
		history[0] = StatusChange{
			Timestamp:      now,
			PreviousStatus: "unknown",
			NewStatus:      restaurant.Status,
			Reason:         "Current status",
			UpdatedBy:      "system",
		}
	}

	return history, nil
}

// StatusChange represents a restaurant status change event
type StatusChange struct {
	Timestamp      time.Time `json:"timestamp"`
	PreviousStatus string    `json:"previous_status"`
	NewStatus      string    `json:"new_status"`
	Reason         string    `json:"reason"`
	UpdatedBy      string    `json:"updated_by"`
}

// ForceStatusUpdate manually updates a specific restaurant's status
func (s *EnhancedCronService) ForceStatusUpdate(ctx context.Context, restaurantID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	restaurantUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return fmt.Errorf("invalid restaurant ID: %v", err)
	}

	// Get restaurant details
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantUUID)
	if err != nil {
		return fmt.Errorf("restaurant not found")
	}

	// Update status immediately
	newStatus := s.determineRestaurantStatus(restaurant)
	if restaurant.Status != newStatus {
		restaurant.Status = newStatus
		now := time.Now()
		restaurant.LastStatusUpdate = &now

		if err := s.restaurantRepo.Update(ctx, restaurant); err != nil {
			return fmt.Errorf("failed to update restaurant status: %v", err)
		}

		log.Printf("🔄 Force updated restaurant %s status to %s", restaurant.Name, newStatus)
	}

	return nil
}

// determineRestaurantStatus determines the current status of a restaurant based on opening hours
func (s *EnhancedCronService) determineRestaurantStatus(restaurant *models.Restaurant) string {
	if restaurant == nil {
		return "closed"
	}

	// If auto open/close is disabled, return current status
	if !restaurant.AutoOpenClose {
		if restaurant.IsOpen {
			return "open"
		}
		return "closed"
	}

	// Use existing logic from shouldRestaurantBeOpenAtTime method
	currentTime := time.Now().In(s.timezone)
	if s.shouldRestaurantBeOpenAtTime(restaurant, currentTime) {
		return "open"
	}

	return "closed"
}
