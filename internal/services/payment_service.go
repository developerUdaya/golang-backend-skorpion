package services

import (
	"context"
	"encoding/json"
	"errors"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type PaymentService struct {
	paymentRepo repositories.PaymentRepository
	orderRepo   repositories.OrderRepository
	cartRepo    repositories.CartRepository
}

func NewPaymentService(
	paymentRepo repositories.PaymentRepository,
	orderRepo repositories.OrderRepository,
	cartRepo repositories.CartRepository,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		orderRepo:   orderRepo,
		cartRepo:    cartRepo,
	}
}

func (s *PaymentService) CreatePayment(payment *models.Payment) error {
	ctx := context.Background()
	return s.paymentRepo.Create(ctx, payment)
}

func (s *PaymentService) GetPaymentByID(id uuid.UUID) (*models.Payment, error) {
	ctx := context.Background()
	return s.paymentRepo.GetByID(ctx, id)
}

func (s *PaymentService) GetPaymentByOrderID(orderID uuid.UUID) (*models.Payment, error) {
	ctx := context.Background()
	return s.paymentRepo.GetByOrderID(ctx, orderID)
}

func (s *PaymentService) UpdatePaymentStatus(id uuid.UUID, status string) (*models.Payment, error) {
	ctx := context.Background()

	// Get the payment first
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update the status
	payment.Status = status

	// Save the updated payment
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) GetPaymentsByStatus(status string, page, limit int) ([]models.Payment, int, error) {
	ctx := context.Background()
	offset := (page - 1) * limit

	payments, err := s.paymentRepo.GetByStatus(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Return with count (simplified for now)
	total := len(payments)
	return payments, total, nil
}

func (s *PaymentService) ProcessWebhook(webhookData interface{}) error {
	ctx := context.Background()

	// Cast webhook data to the expected format
	webhook, ok := webhookData.(*PaymentWebhookRequest)
	if !ok {
		return errors.New("invalid webhook data format")
	}

	// Find payment by transaction ID or payment ID
	var payment *models.Payment
	var err error

	if webhook.TransactionID != "" {
		payment, err = s.paymentRepo.GetByTransactionID(ctx, webhook.TransactionID)
	} else if webhook.PaymentID != "" {
		paymentUUID, parseErr := uuid.Parse(webhook.PaymentID)
		if parseErr != nil {
			return errors.New("invalid payment ID format")
		}
		payment, err = s.paymentRepo.GetByID(ctx, paymentUUID)
	} else {
		return errors.New("payment ID or transaction ID required")
	}

	if err != nil {
		return errors.New("payment not found")
	}

	// Update payment status
	payment.Status = webhook.Status

	// Store webhook metadata
	metadataBytes, _ := json.Marshal(webhook.Metadata)
	var metadata map[string]interface{}
	json.Unmarshal(metadataBytes, &metadata)
	payment.Metadata = metadata

	// Update transaction ID if provided and not already set
	if webhook.TransactionID != "" && payment.TransactionID == "" {
		payment.TransactionID = webhook.TransactionID
	}

	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return err
	}

	// Get the associated order
	order, err := s.orderRepo.GetByID(ctx, payment.OrderID)
	if err != nil {
		return err
	}

	// Handle payment success
	if webhook.Status == "success" || webhook.Status == "completed" {
		// Update order status
		order.OrderStatus = "confirmed"

		// Add order log
		orderLog := map[string]interface{}{
			"timestamp": time.Now(),
			"status":    "payment_completed",
			"message":   "Payment completed successfully",
			"amount":    webhook.Amount,
		}

		var logs []interface{}
		if order.OrderLogs != nil {
			logsBytes, _ := json.Marshal(order.OrderLogs)
			json.Unmarshal(logsBytes, &logs)
		}
		logs = append(logs, orderLog)

		logsMap := make(map[string]interface{})
		logsBytes, _ := json.Marshal(logs)
		json.Unmarshal(logsBytes, &logsMap)
		order.OrderLogs = logsMap

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return err
		}

		// Clear the user's cart
		cart, err := s.cartRepo.GetByUserID(ctx, order.UserID)
		if err == nil {
			// Clear cart items
			cart.Items = models.JSONB{}
			cart.TotalAmount = 0
			cart.Status = "completed"
			cart.UpdatedAt = time.Now()
			s.cartRepo.Update(ctx, cart)
		}

		// TODO: Create delivery partner booking using restaurant's primary delivery partner
		// This would involve calling Porter API or other delivery partner APIs

	} else if webhook.Status == "failed" || webhook.Status == "cancelled" {
		// Handle payment failure
		order.OrderStatus = "payment_failed"

		orderLog := map[string]interface{}{
			"timestamp": time.Now(),
			"status":    "payment_failed",
			"message":   "Payment failed or cancelled",
		}

		var logs []interface{}
		if order.OrderLogs != nil {
			logsBytes, _ := json.Marshal(order.OrderLogs)
			json.Unmarshal(logsBytes, &logs)
		}
		logs = append(logs, orderLog)

		logsMap := make(map[string]interface{})
		logsBytes, _ := json.Marshal(logs)
		json.Unmarshal(logsBytes, &logsMap)
		order.OrderLogs = logsMap

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return err
		}
	}

	return nil
}

type PaymentWebhookRequest struct {
	PaymentID     string                 `json:"payment_id"`
	TransactionID string                 `json:"transaction_id"`
	Status        string                 `json:"status"`
	Amount        float64                `json:"amount"`
	Gateway       string                 `json:"gateway"`
	Metadata      map[string]interface{} `json:"metadata"`
}
