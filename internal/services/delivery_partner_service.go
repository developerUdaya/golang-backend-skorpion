package services

import (
	"context"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type DeliveryPartnerService struct {
	restaurantRepo                repositories.RestaurantRepository
	deliveryPartnerRepo           repositories.DeliveryPartnerRepository
	restaurantDeliveryPartnerRepo repositories.RestaurantDeliveryPartnerRepository
	porterService                 *PorterService
	orderRepo                     repositories.OrderRepository
	porterDeliveryRepo            repositories.PorterDeliveryRepository
}

func NewDeliveryPartnerService(
	restaurantRepo repositories.RestaurantRepository,
	deliveryPartnerRepo repositories.DeliveryPartnerRepository,
	restaurantDeliveryPartnerRepo repositories.RestaurantDeliveryPartnerRepository,
	orderRepo repositories.OrderRepository,
	porterDeliveryRepo repositories.PorterDeliveryRepository,
) *DeliveryPartnerService {
	return &DeliveryPartnerService{
		restaurantRepo:                restaurantRepo,
		deliveryPartnerRepo:           deliveryPartnerRepo,
		restaurantDeliveryPartnerRepo: restaurantDeliveryPartnerRepo,
		porterService:                 NewPorterService(orderRepo, porterDeliveryRepo),
		orderRepo:                     orderRepo,
		porterDeliveryRepo:            porterDeliveryRepo,
	}
}

// CreateDeliveryOrder creates a delivery order with the restaurant's delivery partner
func (s *DeliveryPartnerService) CreateDeliveryOrder(ctx context.Context, order *models.Order) error {
	// Get restaurant details
	restaurant, err := s.restaurantRepo.GetByID(ctx, order.RestaurantID)
	if err != nil {
		return fmt.Errorf("failed to get restaurant: %v", err)
	}

	// Get restaurant's delivery partners
	deliveryPartners, err := s.restaurantDeliveryPartnerRepo.GetByRestaurantID(ctx, order.RestaurantID)
	if err != nil {
		return fmt.Errorf("failed to get delivery partners: %v", err)
	}

	if len(deliveryPartners) == 0 {
		return fmt.Errorf("no delivery partners found for restaurant %s", restaurant.Name)
	}

	// For now, use the first delivery partner (Porter)
	deliveryPartner := deliveryPartners[0]

	// Get delivery partner company details
	partnerCompany, err := s.deliveryPartnerRepo.GetByID(ctx, deliveryPartner.DeliveryPartnerCompanyID)
	if err != nil {
		return fmt.Errorf("failed to get delivery partner company: %v", err)
	}

	// Use Porter service for Porter delivery partners
	if partnerCompany.Name == "Porter" {
		porterOrder, err := s.porterService.CreateDeliveryOrder(ctx, order, restaurant)
		if err != nil {
			return fmt.Errorf("failed to create Porter delivery order: %v", err)
		}

		// Update order with delivery partner info
		order.DeliveryPartnerID = &deliveryPartner.ID

		// Log successful delivery order creation
		fmt.Printf("Porter delivery order created successfully:\n")
		fmt.Printf("- Order ID: %s\n", order.ID.String())
		fmt.Printf("- Porter Order ID: %s\n", porterOrder.OrderID)
		fmt.Printf("- Delivery Partner: %s\n", partnerCompany.Name)
		fmt.Printf("- Tracking URL: %s\n", porterOrder.TrackingURL)
		fmt.Printf("- Estimated Pickup Time: %d\n", porterOrder.EstimatedPickupTime)
		fmt.Printf("- Estimated Fare: ₹%.2f\n", float64(porterOrder.EstimatedFareDetails.MinorAmount)/100)

		return nil
	}

	// For other delivery partners, use the legacy mock implementation
	return s.createLegacyDeliveryOrder(ctx, order, restaurant, partnerCompany, deliveryPartner)
}

// createLegacyDeliveryOrder handles non-Porter delivery partners (mock implementation)
func (s *DeliveryPartnerService) createLegacyDeliveryOrder(ctx context.Context, order *models.Order, restaurant *models.Restaurant, partnerCompany *models.DeliveryPartnerCompany, deliveryPartner models.RestaurantDeliveryPartners) error {
	// Mock implementation for other delivery partners
	// Update order with delivery partner info
	order.DeliveryPartnerID = &deliveryPartner.ID

	// Add order log
	orderLog := map[string]interface{}{
		"action":           "delivery_order_created",
		"delivery_partner": partnerCompany.Name,
		"order_id":         fmt.Sprintf("%s_%d", partnerCompany.Name, time.Now().Unix()),
		"estimated_time":   30,   // Mock 30 minutes
		"delivery_fee":     45.0, // Mock ₹45 delivery fee
		"timestamp":        time.Now().Unix(),
	}

	if order.OrderLogs == nil {
		order.OrderLogs = make(models.JSONB)
	}

	// Add to existing order logs
	logs, ok := order.OrderLogs["logs"].([]interface{})
	if !ok {
		logs = []interface{}{}
	}
	logs = append(logs, orderLog)
	order.OrderLogs["logs"] = logs

	fmt.Printf("Legacy delivery order created successfully:\n")
	fmt.Printf("- Order ID: %s\n", order.ID.String())
	fmt.Printf("- Delivery Partner: %s\n", partnerCompany.Name)
	fmt.Printf("- Estimated Time: %d minutes\n", 30)
	fmt.Printf("- Delivery Fee: ₹%.2f\n", 45.0)

	return nil
}

// ReassignDeliveryPartner reassigns delivery partner for an order
func (s *DeliveryPartnerService) ReassignDeliveryPartner(ctx context.Context, orderID uuid.UUID, preferredPartner string) error {
	// Get the order
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %v", err)
	}

	// If requesting Porter reassignment
	if preferredPartner == "porter" {
		// Deactivate old Porter deliveries
		if err := s.porterDeliveryRepo.DeactivateOldDeliveries(ctx, orderID); err != nil {
			return fmt.Errorf("failed to deactivate old deliveries: %v", err)
		}

		// Cancel existing active Porter delivery if any
		activeDelivery, err := s.porterDeliveryRepo.GetActiveByOrderID(ctx, orderID)
		if err == nil && activeDelivery != nil {
			// Try to cancel the old Porter order
			_, cancelErr := s.porterService.CancelOrder(ctx, activeDelivery.PorterOrderID)
			if cancelErr != nil {
				// Log error but continue - the order might already be in a non-cancellable state
				fmt.Printf("Warning: Failed to cancel old Porter order %s: %v\n", activeDelivery.PorterOrderID, cancelErr)
			}
		}

		// Create new Porter delivery
		return s.CreateDeliveryOrder(ctx, order)
	}

	// For other delivery partners, implement similar logic
	return fmt.Errorf("reassignment for partner %s not yet implemented", preferredPartner)
}
