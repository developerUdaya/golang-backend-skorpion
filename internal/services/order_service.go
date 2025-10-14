package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/cache"
	"golang-food-backend/pkg/messaging"
	"time"

	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo     repositories.OrderRepository
	cartRepo      repositories.CartRepository
	paymentRepo   repositories.PaymentRepository
	userRepo      repositories.UserRepository
	inventoryRepo repositories.InventoryRepository
	cache         *cache.RedisCache
	kafkaProducer *messaging.KafkaProducer
	kafkaBrokers  []string
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	cartRepo repositories.CartRepository,
	paymentRepo repositories.PaymentRepository,
	userRepo repositories.UserRepository,
	inventoryRepo repositories.InventoryRepository,
	cache *cache.RedisCache,
	kafkaProducer *messaging.KafkaProducer,
	kafkaBrokers []string,
) *OrderService {
	return &OrderService{
		orderRepo:     orderRepo,
		cartRepo:      cartRepo,
		paymentRepo:   paymentRepo,
		userRepo:      userRepo,
		inventoryRepo: inventoryRepo,
		cache:         cache,
		kafkaProducer: kafkaProducer,
		kafkaBrokers:  kafkaBrokers,
	}
}

type CreateOrderRequest struct {
	CartID                         string                 `json:"cart_id" binding:"required"`
	AddressID                      *string                `json:"address_id,omitempty"`
	PaymentMethod                  string                 `json:"payment_method" binding:"required"`
	DeliveryFullAddressWithLatLong map[string]interface{} `json:"delivery_full_address_with_lat_long"`
	CustomerName                   string                 `json:"customer_name" binding:"required"`
	CustomerContact                string                 `json:"customer_contact" binding:"required"`
	CouponCode                     *string                `json:"coupon_code,omitempty"`
}

type OrderResponse struct {
	Order      *models.Order   `json:"order"`
	Payment    *models.Payment `json:"payment,omitempty"`
	PaymentURL string          `json:"payment_url,omitempty"`
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, req *CreateOrderRequest) (*OrderResponse, error) {
	// Parse cart ID
	cartUUID, err := uuid.Parse(req.CartID)
	if err != nil {
		return nil, errors.New("invalid cart ID")
	}

	// Get cart
	cart, err := s.cartRepo.GetByID(ctx, cartUUID)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	// Verify cart belongs to user
	if cart.UserID.String() != userID {
		return nil, errors.New("cart does not belong to user")
	}

	// Verify cart is active
	if cart.Status != "active" {
		return nil, errors.New("cart is not active")
	}

	// Parse cart items
	var cartItems []models.CartItem
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", cart.Items)), &cartItems); err != nil {
		return nil, errors.New("invalid cart items")
	}

	if len(cartItems) == 0 {
		return nil, errors.New("cart is empty")
	}

	// Reserve inventory for each item
	for range cartItems {
		// This would require converting product ID string to ObjectID and reserving stock
		// For now, we'll simulate this
	}

	// Parse user and address IDs
	userUUID, _ := uuid.Parse(userID)
	var addressUUID *uuid.UUID
	if req.AddressID != nil {
		addr, err := uuid.Parse(*req.AddressID)
		if err == nil {
			addressUUID = &addr
		}
	}

	// Create order
	order := &models.Order{
		UserID:                         userUUID,
		RestaurantID:                   cart.RestaurantID,
		CartID:                         cartUUID,
		OrderStatus:                    "pending",
		TotalAmount:                    cart.TotalAmount,
		CustomerName:                   req.CustomerName,
		CustomerContact:                req.CustomerContact,
		AddressID:                      addressUUID,
		DeliveryFullAddressWithLatLong: req.DeliveryFullAddressWithLatLong,
		CreatedAt:                      time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Create payment record
	payment := &models.Payment{
		OrderID:   order.ID,
		UserID:    userUUID,
		Amount:    order.TotalAmount,
		Method:    req.PaymentMethod,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Update order with payment ID
	order.PaymentID = &payment.ID
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Mark cart as used
	cart.Status = "ordered"
	if err := s.cartRepo.Update(ctx, cart); err != nil {
		return nil, err
	}

	// Send order event to Kafka
	orderEvent := messaging.OrderEvent{
		Type:    "order_created",
		OrderID: order.ID.String(),
		UserID:  userID,
		Data:    order,
	}
	s.kafkaProducer.SendMessage("order_events", s.kafkaBrokers, order.ID.String(), orderEvent)

	// Send notification event
	notificationEvent := messaging.NotificationEvent{
		Type:    "order_confirmation",
		UserID:  userID,
		Title:   "Order Confirmed",
		Message: fmt.Sprintf("Your order #%s has been confirmed", order.ID.String()[:8]),
		Metadata: map[string]interface{}{
			"order_id": order.ID.String(),
			"amount":   order.TotalAmount,
		},
	}
	s.kafkaProducer.SendMessage("notification_events", s.kafkaBrokers, userID, notificationEvent)

	response := &OrderResponse{
		Order:   order,
		Payment: payment,
	}

	// For non-cash payments, generate payment URL
	if req.PaymentMethod != "cash" {
		response.PaymentURL = s.generatePaymentURL(payment)
	}

	return response, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string, userID string) (*models.Order, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, errors.New("invalid order ID")
	}

	order, err := s.orderRepo.GetByID(ctx, orderUUID)
	if err != nil {
		return nil, err
	}

	// Verify order belongs to user (unless admin/restaurant staff)
	if order.UserID.String() != userID {
		// Here you'd check if the user has permission to view this order
		return nil, errors.New("order not found")
	}

	return order, nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]models.Order, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	return s.orderRepo.GetByUserID(ctx, userUUID, limit, offset)
}

func (s *OrderService) GetRestaurantOrders(ctx context.Context, restaurantID string, limit, offset int) ([]models.Order, error) {
	restUUID, err := uuid.Parse(restaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	return s.orderRepo.GetByRestaurantID(ctx, restUUID, limit, offset)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus string, restaurantID string) error {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return errors.New("invalid order ID")
	}

	order, err := s.orderRepo.GetByID(ctx, orderUUID)
	if err != nil {
		return err
	}

	// Verify order belongs to restaurant
	if order.RestaurantID.String() != restaurantID {
		return errors.New("order does not belong to this restaurant")
	}

	// Validate status transition
	if !s.isValidStatusTransition(order.OrderStatus, newStatus) {
		return errors.New("invalid status transition")
	}

	// Update order status
	order.OrderStatus = newStatus

	// Add to order logs
	logEntry := map[string]interface{}{
		"timestamp": time.Now(),
		"status":    newStatus,
		"note":      fmt.Sprintf("Status updated to %s", newStatus),
	}

	// Handle OrderLogs as JSONB (map[string]interface{})
	if order.OrderLogs == nil {
		order.OrderLogs = models.JSONB{}
	}

	// Get existing logs
	var logs []map[string]interface{}
	if existingLogs, ok := order.OrderLogs["logs"]; ok {
		if logSlice, ok := existingLogs.([]interface{}); ok {
			for _, log := range logSlice {
				if logMap, ok := log.(map[string]interface{}); ok {
					logs = append(logs, logMap)
				}
			}
		}
	}

	// Append new log entry
	logs = append(logs, logEntry)
	order.OrderLogs["logs"] = logs

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	// Send status update event
	orderEvent := messaging.OrderEvent{
		Type:    "order_status_updated",
		OrderID: order.ID.String(),
		UserID:  order.UserID.String(),
		Data: map[string]interface{}{
			"order_id":   order.ID.String(),
			"new_status": newStatus,
			"old_status": order.OrderStatus,
		},
	}
	s.kafkaProducer.SendMessage("order_events", s.kafkaBrokers, order.ID.String(), orderEvent)

	// Send notification to user
	notificationEvent := messaging.NotificationEvent{
		Type:    "order_status_update",
		UserID:  order.UserID.String(),
		Title:   "Order Update",
		Message: s.getStatusUpdateMessage(newStatus),
		Metadata: map[string]interface{}{
			"order_id": order.ID.String(),
			"status":   newStatus,
		},
	}
	s.kafkaProducer.SendMessage("notification_events", s.kafkaBrokers, order.UserID.String(), notificationEvent)

	return nil
}

func (s *OrderService) isValidStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		"pending":    {"confirmed", "cancelled"},
		"confirmed":  {"preparing", "cancelled"},
		"preparing":  {"dispatched", "cancelled"},
		"dispatched": {"delivered", "cancelled"},
		"delivered":  {}, // terminal state
		"cancelled":  {}, // terminal state
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowedStatuses {
		if newStatus == allowedStatus {
			return true
		}
	}

	return false
}

func (s *OrderService) getStatusUpdateMessage(status string) string {
	messages := map[string]string{
		"confirmed":  "Your order has been confirmed by the restaurant",
		"preparing":  "Your order is being prepared",
		"dispatched": "Your order is on the way",
		"delivered":  "Your order has been delivered",
		"cancelled":  "Your order has been cancelled",
	}

	if message, exists := messages[status]; exists {
		return message
	}
	return "Your order status has been updated"
}

func (s *OrderService) generatePaymentURL(payment *models.Payment) string {
	// This would integrate with payment gateways like Razorpay, Stripe, etc.
	// For now, return a mock URL
	return fmt.Sprintf("https://payment.gateway.com/pay/%s", payment.ID.String())
}
