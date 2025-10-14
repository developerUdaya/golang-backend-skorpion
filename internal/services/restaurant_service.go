package services

import (
	"context"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"

	"github.com/google/uuid"
)

type RestaurantService struct {
	restaurantRepo repositories.RestaurantRepository
}

func NewRestaurantService(restaurantRepo repositories.RestaurantRepository) *RestaurantService {
	return &RestaurantService{
		restaurantRepo: restaurantRepo,
	}
}

func (s *RestaurantService) CreateRestaurant(restaurant *models.Restaurant) error {
	ctx := context.Background()
	return s.restaurantRepo.Create(ctx, restaurant)
}

func (s *RestaurantService) GetRestaurants(page, limit int, cuisine, search string) ([]models.Restaurant, int, error) {
	ctx := context.Background()
	offset := (page - 1) * limit

	// For now, just use the search method from interface
	restaurants, err := s.restaurantRepo.Search(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Return with count (simplified for now)
	total := len(restaurants)
	return restaurants, total, nil
}

func (s *RestaurantService) GetRestaurantByID(id uuid.UUID) (*models.Restaurant, error) {
	ctx := context.Background()
	return s.restaurantRepo.GetByID(ctx, id)
}

func (s *RestaurantService) UpdateRestaurant(restaurant *models.Restaurant) error {
	ctx := context.Background()
	return s.restaurantRepo.Update(ctx, restaurant)
}

func (s *RestaurantService) DeleteRestaurant(id uuid.UUID) error {
	ctx := context.Background()
	return s.restaurantRepo.Delete(ctx, id)
}

func (s *RestaurantService) GetRestaurantsByOwner(ownerID uuid.UUID) ([]models.Restaurant, error) {
	ctx := context.Background()
	return s.restaurantRepo.GetByOwnerID(ctx, ownerID)
}
