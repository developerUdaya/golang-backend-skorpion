package services

import (
	"context"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"

	"github.com/google/uuid"
)

type PaymentService struct {
	paymentRepo repositories.PaymentRepository
}

func NewPaymentService(paymentRepo repositories.PaymentRepository) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
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
	// Process webhook data and update payment status
	// This would typically involve validating the webhook signature
	// and updating the corresponding payment record

	// For now, just return nil (implementation depends on payment gateway)
	return nil
}
