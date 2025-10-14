package services

import (
	"context"
	"errors"
	"time"

	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"

	"github.com/google/uuid"
)

type CouponService struct {
	couponRepo repositories.CouponRepository
}

func NewCouponService(couponRepo repositories.CouponRepository) *CouponService {
	return &CouponService{
		couponRepo: couponRepo,
	}
}

// Request and Response types
type CreateCouponRequest struct {
	Code                  string   `json:"code" binding:"required"`
	Description           string   `json:"description" binding:"required"`
	DiscountType          string   `json:"discount_type" binding:"required,oneof=percentage fixed"`
	DiscountValue         float64  `json:"discount_value" binding:"required,min=0"`
	MinimumOrderAmount    *float64 `json:"minimum_order_amount"`
	MaximumDiscountAmount *float64 `json:"maximum_discount_amount"`
	UsageLimit            *int     `json:"usage_limit"`
	ValidFrom             string   `json:"valid_from" binding:"required"`
	ValidUntil            string   `json:"valid_until" binding:"required"`
	RestaurantID          *string  `json:"restaurant_id"`
	IsActive              bool     `json:"is_active"`
}

type UpdateCouponRequest struct {
	Description           string   `json:"description"`
	DiscountType          string   `json:"discount_type" binding:"omitempty,oneof=percentage fixed"`
	DiscountValue         *float64 `json:"discount_value" binding:"omitempty,min=0"`
	MinimumOrderAmount    *float64 `json:"minimum_order_amount"`
	MaximumDiscountAmount *float64 `json:"maximum_discount_amount"`
	UsageLimit            *int     `json:"usage_limit"`
	ValidFrom             string   `json:"valid_from"`
	ValidUntil            string   `json:"valid_until"`
	IsActive              *bool    `json:"is_active"`
}

type ValidateCouponRequest struct {
	Code         string  `json:"code" binding:"required"`
	RestaurantID string  `json:"restaurant_id" binding:"required"`
	OrderAmount  float64 `json:"order_amount" binding:"required,min=0"`
}

type CouponListResponse struct {
	Coupons    []models.Coupon `json:"coupons"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	TotalPages int             `json:"total_pages"`
}

type CouponValidationResponse struct {
	Valid          bool           `json:"valid"`
	DiscountType   string         `json:"discount_type,omitempty"`
	DiscountValue  float64        `json:"discount_value,omitempty"`
	DiscountAmount float64        `json:"discount_amount,omitempty"`
	Message        string         `json:"message,omitempty"`
	Coupon         *models.Coupon `json:"coupon,omitempty"`
}

func (s *CouponService) CreateCoupon(ctx context.Context, userID string, req *CreateCouponRequest) (*models.Coupon, error) {
	// Parse dates
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		return nil, errors.New("invalid valid_from date format")
	}

	validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		return nil, errors.New("invalid valid_until date format")
	}

	if validUntil.Before(validFrom) {
		return nil, errors.New("valid_until must be after valid_from")
	}

	// Create coupon
	coupon := &models.Coupon{
		Code:          req.Code,
		Description:   req.Description,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		ValidFrom:     validFrom,
		ValidTo:       validUntil,
		IsActive:      req.IsActive,
	}

	// Set optional fields
	if req.MinimumOrderAmount != nil {
		coupon.MinOrderValue = *req.MinimumOrderAmount
	}
	if req.MaximumDiscountAmount != nil {
		coupon.MaxDiscount = *req.MaximumDiscountAmount
	}
	if req.UsageLimit != nil {
		coupon.UsageLimit = *req.UsageLimit
	} else {
		coupon.UsageLimit = -1 // unlimited
	}

	// Set restaurant ID if provided
	if req.RestaurantID != nil {
		restID, err := uuid.Parse(*req.RestaurantID)
		if err != nil {
			return nil, errors.New("invalid restaurant ID")
		}
		coupon.RestaurantID = &restID
	}

	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		return nil, err
	}

	return coupon, nil
}

func (s *CouponService) GetCoupons(ctx context.Context, page, limit int, restaurantID string, active *bool) (*CouponListResponse, error) {
	offset := (page - 1) * limit

	var restID *uuid.UUID
	if restaurantID != "" {
		parsed, err := uuid.Parse(restaurantID)
		if err != nil {
			return nil, errors.New("invalid restaurant ID")
		}
		restID = &parsed
	}

	coupons, total, err := s.couponRepo.GetCouponsWithFilters(ctx, offset, limit, restID, active)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &CouponListResponse{
		Coupons:    coupons,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

func (s *CouponService) GetCouponByID(ctx context.Context, couponID string) (*models.Coupon, error) {
	id, err := uuid.Parse(couponID)
	if err != nil {
		return nil, errors.New("invalid coupon ID")
	}

	return s.couponRepo.GetByID(ctx, id)
}

func (s *CouponService) ValidateCoupon(ctx context.Context, userID string, req *ValidateCouponRequest) (*CouponValidationResponse, error) {
	var restID *uuid.UUID
	if req.RestaurantID != "" {
		parsed, err := uuid.Parse(req.RestaurantID)
		if err != nil {
			return &CouponValidationResponse{
				Valid:   false,
				Message: "Invalid restaurant ID",
			}, nil
		}
		restID = &parsed
	}

	// Get coupon by code
	coupon, err := s.couponRepo.GetByCode(ctx, req.Code)
	if err != nil {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Coupon not found",
		}, nil
	}

	// Check if coupon is active
	if !coupon.IsActive {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Coupon is not active",
		}, nil
	}

	// Check if coupon is valid for this restaurant
	if coupon.RestaurantID != nil && restID != nil && *coupon.RestaurantID != *restID {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Coupon is not valid for this restaurant",
		}, nil
	}

	// Check date validity
	now := time.Now()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidTo) {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Coupon has expired or is not yet valid",
		}, nil
	}

	// Check usage limit
	if coupon.UsageLimit > 0 && coupon.UsedCount >= coupon.UsageLimit {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Coupon usage limit exceeded",
		}, nil
	}

	// Check minimum order amount
	if coupon.MinOrderValue > 0 && req.OrderAmount < coupon.MinOrderValue {
		return &CouponValidationResponse{
			Valid:   false,
			Message: "Order amount is below minimum required",
		}, nil
	}

	// Calculate discount amount
	var discountAmount float64
	if coupon.DiscountType == "percentage" {
		discountAmount = req.OrderAmount * (coupon.DiscountValue / 100)
		if coupon.MaxDiscount > 0 && discountAmount > coupon.MaxDiscount {
			discountAmount = coupon.MaxDiscount
		}
	} else {
		discountAmount = coupon.DiscountValue
	}

	return &CouponValidationResponse{
		Valid:          true,
		DiscountType:   coupon.DiscountType,
		DiscountValue:  coupon.DiscountValue,
		DiscountAmount: discountAmount,
		Message:        "Coupon is valid",
		Coupon:         coupon,
	}, nil
}

func (s *CouponService) UpdateCoupon(ctx context.Context, userID, couponID string, req *UpdateCouponRequest) (*models.Coupon, error) {
	id, err := uuid.Parse(couponID)
	if err != nil {
		return nil, errors.New("invalid coupon ID")
	}

	coupon, err := s.couponRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.DiscountType != "" {
		coupon.DiscountType = req.DiscountType
	}
	if req.DiscountValue != nil {
		coupon.DiscountValue = *req.DiscountValue
	}
	if req.MinimumOrderAmount != nil {
		coupon.MinOrderValue = *req.MinimumOrderAmount
	}
	if req.MaximumDiscountAmount != nil {
		coupon.MaxDiscount = *req.MaximumDiscountAmount
	}
	if req.UsageLimit != nil {
		coupon.UsageLimit = *req.UsageLimit
	}
	if req.ValidFrom != "" {
		validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
		if err != nil {
			return nil, errors.New("invalid valid_from date format")
		}
		coupon.ValidFrom = validFrom
	}
	if req.ValidUntil != "" {
		validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
		if err != nil {
			return nil, errors.New("invalid valid_until date format")
		}
		coupon.ValidTo = validUntil
	}
	if req.IsActive != nil {
		coupon.IsActive = *req.IsActive
	}

	if err := s.couponRepo.Update(ctx, coupon); err != nil {
		return nil, err
	}

	return coupon, nil
}

func (s *CouponService) DeleteCoupon(ctx context.Context, userID, couponID string) error {
	id, err := uuid.Parse(couponID)
	if err != nil {
		return errors.New("invalid coupon ID")
	}

	coupon, err := s.couponRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Soft delete by marking as inactive
	coupon.IsActive = false

	return s.couponRepo.Update(ctx, coupon)
}
