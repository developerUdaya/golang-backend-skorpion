package services

import (
	"context"
	"errors"
	"time"

	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"

	"github.com/google/uuid"
)

type RefundService struct {
	refundRepo  repositories.RefundRepository
	orderRepo   repositories.OrderRepository
	paymentRepo repositories.PaymentRepository
}

func NewRefundService(
	refundRepo repositories.RefundRepository,
	orderRepo repositories.OrderRepository,
	paymentRepo repositories.PaymentRepository,
) *RefundService {
	return &RefundService{
		refundRepo:  refundRepo,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
	}
}

// Request and Response types
type CreateRefundRequest struct {
	OrderID string  `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required,min=0"`
	Reason  string  `json:"reason" binding:"required"`
}

type UpdateRefundStatusRequest struct {
	Status       string `json:"status" binding:"required,oneof=pending approved rejected processed failed"`
	AdminComment string `json:"admin_comment"`
}

type RefundListResponse struct {
	Refunds    []models.Refund `json:"refunds"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	TotalPages int             `json:"total_pages"`
}

func (s *RefundService) CreateRefund(ctx context.Context, userID string, req *CreateRefundRequest) (*models.Refund, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return nil, errors.New("invalid order ID")
	}

	// Verify order belongs to user and is eligible for refund
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if order.UserID != userUUID {
		return nil, errors.New("order does not belong to user")
	}

	// Check if order is in a refundable status
	if order.OrderStatus != "completed" && order.OrderStatus != "delivered" {
		return nil, errors.New("order is not eligible for refund")
	}

	// Check if refund amount is valid
	if req.Amount > order.TotalAmount {
		return nil, errors.New("refund amount cannot exceed order total")
	}

	// Check if there's already a refund for this order
	existing, _ := s.refundRepo.GetByOrderID(ctx, orderID)
	if existing != nil {
		return nil, errors.New("refund request already exists for this order")
	}

	// Create refund record
	refund := &models.Refund{
		OrderID: orderID,
		Amount:  req.Amount,
		Reason:  req.Reason,
		Status:  "pending",
	}

	if err := s.refundRepo.Create(ctx, refund); err != nil {
		return nil, err
	}

	return refund, nil
}

func (s *RefundService) GetRefunds(ctx context.Context, userID string, page, limit int, status string) (*RefundListResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	offset := (page - 1) * limit

	refunds, total, err := s.refundRepo.GetByUserIDWithFilters(ctx, userUUID, offset, limit, status)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &RefundListResponse{
		Refunds:    refunds,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

func (s *RefundService) GetRefundByID(ctx context.Context, userID, refundID string) (*models.Refund, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	id, err := uuid.Parse(refundID)
	if err != nil {
		return nil, errors.New("invalid refund ID")
	}

	refund, err := s.refundRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify refund belongs to user (through order)
	order, err := s.orderRepo.GetByID(ctx, refund.OrderID)
	if err != nil {
		return nil, errors.New("associated order not found")
	}

	if order.UserID != userUUID {
		return nil, errors.New("refund does not belong to user")
	}

	return refund, nil
}

func (s *RefundService) UpdateRefundStatus(ctx context.Context, adminID, refundID string, req *UpdateRefundStatusRequest) (*models.Refund, error) {
	id, err := uuid.Parse(refundID)
	if err != nil {
		return nil, errors.New("invalid refund ID")
	}

	refund, err := s.refundRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if !isValidStatusTransition(refund.Status, req.Status) {
		return nil, errors.New("invalid status transition")
	}

	// Update refund
	refund.Status = req.Status
	if req.AdminComment != "" {
		refund.AdminComment = &req.AdminComment
	}

	if err := s.refundRepo.Update(ctx, refund); err != nil {
		return nil, err
	}

	return refund, nil
}

func (s *RefundService) ProcessRefund(ctx context.Context, adminID, refundID string) (*models.Refund, error) {
	id, err := uuid.Parse(refundID)
	if err != nil {
		return nil, errors.New("invalid refund ID")
	}

	refund, err := s.refundRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if refund.Status != "approved" {
		return nil, errors.New("refund must be approved before processing")
	}

	// Get the original payment to refund
	_, err = s.paymentRepo.GetByOrderID(ctx, refund.OrderID)
	if err != nil {
		return nil, errors.New("original payment not found")
	}

	// TODO: Integrate with payment gateway to process the actual refund
	// For now, we'll mark it as processed

	refund.Status = "processed"
	refund.ProcessedAt = &time.Time{}
	*refund.ProcessedAt = time.Now()

	if err := s.refundRepo.Update(ctx, refund); err != nil {
		return nil, err
	}

	// TODO: Update payment status and create refund transaction record

	return refund, nil
}

func isValidStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		"pending":   {"approved", "rejected"},
		"approved":  {"processed", "failed"},
		"rejected":  {},           // No further transitions
		"processed": {},           // No further transitions
		"failed":    {"approved"}, // Can retry
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}

	return false
}
