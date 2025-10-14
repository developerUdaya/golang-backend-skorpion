package services

import (
	"context"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"log"
	"time"
)

type CronService struct {
	ticker          *time.Ticker
	stopChan        chan bool
	restaurantRepo  repositories.RestaurantRepository
	shopTimeService *ShopTimeService
}

func NewCronService(
	restaurantRepo repositories.RestaurantRepository,
	shopTimeService *ShopTimeService,
) *CronService {
	return &CronService{
		stopChan:        make(chan bool),
		restaurantRepo:  restaurantRepo,
		shopTimeService: shopTimeService,
	}
}

func (s *CronService) Start() error {
	// Run every minute to check restaurant status
	s.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.updateAllRestaurantStatus()
			case <-s.stopChan:
				return
			}
		}
	}()

	log.Println("Cron service started - Auto restaurant status management enabled")

	return nil
}

func (s *CronService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
	log.Println("Cron service stopped")
}

func (s *CronService) updateAllRestaurantStatus() {
	ctx := context.Background()
	currentTime := time.Now()

	// In a real implementation, you would paginate through all restaurants
	// For now, we'll create a simple version

	log.Printf("Running auto status update at %s", currentTime.Format("15:04:05"))

	// This would get all restaurants with auto_open_close = true
	// For demonstration, we'll simulate the process
	s.simulateStatusUpdate(ctx, currentTime)
}

func (s *CronService) simulateStatusUpdate(ctx context.Context, currentTime time.Time) {
	currentTimeStr := currentTime.Format("15:04")
	currentDay := currentTime.Weekday().String()

	log.Printf("Checking restaurant status for %s at %s", currentDay, currentTimeStr)

	// In production, this would:
	// 1. Query all restaurants with auto_open_close = true
	// 2. Check their opening hours for current day
	// 3. Update is_open status based on current time
	// 4. Log status changes

	// Example logic (this would be in a loop for all restaurants):
	/*
		restaurants, err := s.restaurantRepo.GetRestaurantsWithAutoOpenClose(ctx)
		if err != nil {
			log.Printf("Error getting restaurants: %v", err)
			return
		}

		for _, restaurant := range restaurants {
			shouldBeOpen := s.shouldRestaurantBeOpen(restaurant, currentTime)

			if restaurant.IsOpen != shouldBeOpen {
				restaurant.IsOpen = shouldBeOpen

				if err := s.restaurantRepo.Update(ctx, &restaurant); err != nil {
					log.Printf("Failed to update restaurant %s status: %v", restaurant.Name, err)
					continue
				}

				status := "CLOSED"
				if shouldBeOpen {
					status = "OPENED"
				}

				log.Printf("Restaurant '%s' automatically %s at %s", restaurant.Name, status, currentTimeStr)
			}
		}
	*/
}

func (s *CronService) shouldRestaurantBeOpen(restaurant models.Restaurant, checkTime time.Time) bool {
	if !restaurant.AutoOpenClose {
		return restaurant.IsOpen
	}

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

	// Handle overnight restaurants (e.g., open 22:00, close 04:00)
	if closeTime < openTime {
		return currentTimeStr >= openTime || currentTimeStr <= closeTime
	}

	return currentTimeStr >= openTime && currentTimeStr <= closeTime
}

// Manual trigger for testing
func (s *CronService) TriggerStatusUpdate() {
	s.updateAllRestaurantStatus()
}
