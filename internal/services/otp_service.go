package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/auth"
	"golang-food-backend/pkg/cache"
	"golang-food-backend/pkg/sms"
	"math/big"
	"time"

	"github.com/google/uuid"
)

type OTPService struct {
	otpRepo    repositories.OTPRepository
	userRepo   repositories.UserRepository
	jwtManager *auth.JWTManager
	cache      *cache.RedisCache
	smsService *sms.SMSService
}

type SendOTPRequest struct {
	Phone        string `json:"phone" binding:"required"`
	Role         string `json:"role" binding:"required"` // Required to determine OTP flow
	RestaurantID string `json:"restaurant_id"`           // Required only for customers
}

type VerifyOTPRequest struct {
	Phone        string `json:"phone" binding:"required"`
	Role         string `json:"role" binding:"required"` // Required to determine login flow
	RestaurantID string `json:"restaurant_id"`           // Required only for customers
	OTPCode      string `json:"otp_code" binding:"required"`
}

type OTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewOTPService(otpRepo repositories.OTPRepository, userRepo repositories.UserRepository, jwtManager *auth.JWTManager, cache *cache.RedisCache, smsService *sms.SMSService) *OTPService {
	return &OTPService{
		otpRepo:    otpRepo,
		userRepo:   userRepo,
		jwtManager: jwtManager,
		cache:      cache,
		smsService: smsService,
	}
}

// generateOTP generates a 6-digit OTP
func (s *OTPService) generateOTP() (string, error) {
	max := big.NewInt(999999)
	min := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, new(big.Int).Sub(max, min))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", new(big.Int).Add(n, min).Int64()), nil
}

func (s *OTPService) SendOTP(ctx context.Context, req *SendOTPRequest) (*OTPResponse, error) {
	var restaurantID *uuid.UUID

	// Validate role and restaurant ID requirements
	if req.Role == "customer" {
		if req.RestaurantID == "" {
			return nil, errors.New("restaurant ID is required for customers")
		}
		parsedRestaurantID, err := uuid.Parse(req.RestaurantID)
		if err != nil {
			return nil, errors.New("invalid restaurant ID")
		}
		restaurantID = &parsedRestaurantID
	} else {
		// For non-customer roles, restaurant ID is optional
		if req.RestaurantID != "" {
			parsedRestaurantID, err := uuid.Parse(req.RestaurantID)
			if err != nil {
				return nil, errors.New("invalid restaurant ID")
			}
			restaurantID = &parsedRestaurantID
		}
	}

	// Generate OTP
	otpCode, err := s.generateOTP()
	if err != nil {
		return nil, errors.New("failed to generate OTP")
	}

	// Create OTP record
	otp := &models.OTP{
		Phone:        req.Phone,
		RestaurantID: restaurantID,
		OTPCode:      otpCode,
		ExpiresAt:    time.Now().Add(5 * time.Minute), // 5 minutes expiry
		IsUsed:       false,
		AttemptCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save OTP to database
	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return nil, errors.New("failed to save OTP")
	}

	// Send SMS
	if err := s.smsService.SendOTP(req.Phone, otpCode); err != nil {
		// Log error but don't fail the request
		// In production, you might want to handle this differently
		fmt.Printf("Failed to send SMS: %v\n", err)
	}

	return &OTPResponse{
		Success: true,
		Message: "OTP sent successfully",
	}, nil
}

func (s *OTPService) VerifyOTPAndLogin(ctx context.Context, req *VerifyOTPRequest) (*AuthResponse, error) {
	var restaurantID *uuid.UUID
	var user *models.User
	var err error

	// Validate role and restaurant ID requirements
	if req.Role == "customer" {
		if req.RestaurantID == "" {
			return nil, errors.New("restaurant ID is required for customer login")
		}
		parsedRestaurantID, err := uuid.Parse(req.RestaurantID)
		if err != nil {
			return nil, errors.New("invalid restaurant ID")
		}
		restaurantID = &parsedRestaurantID
	} else {
		// For non-customer roles, restaurant ID is optional
		if req.RestaurantID != "" {
			parsedRestaurantID, err := uuid.Parse(req.RestaurantID)
			if err != nil {
				return nil, errors.New("invalid restaurant ID")
			}
			restaurantID = &parsedRestaurantID
		}
	}

	// Get valid OTP
	otp, err := s.otpRepo.GetValidOTPWithOptionalRestaurant(ctx, req.Phone, restaurantID, req.OTPCode)
	if err != nil {
		return nil, errors.New("invalid or expired OTP")
	}

	// Role-based user lookup and creation logic
	if req.Role == "customer" {
		// For customers: check within restaurant
		user, err = s.userRepo.GetByPhoneAndRestaurant(ctx, req.Phone, *restaurantID)
		if err != nil {
			// Customer doesn't exist, create a new customer
			user = &models.User{
				Name:         fmt.Sprintf("User-%s", req.Phone[len(req.Phone)-4:]), // Use last 4 digits
				Phone:        req.Phone,
				Email:        "", // Email can be empty for OTP login
				PasswordHash: "", // No password for OTP users
				RestaurantID: restaurantID,
				Role:         "customer",
				Status:       "active",
				IsVerified:   true, // OTP verified means phone is verified
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, errors.New("failed to create user")
			}
		} else {
			// Verify role matches
			if user.Role != "customer" {
				return nil, errors.New("invalid role for this login")
			}
		}
	} else {
		// For non-customer roles: global lookup
		user, err = s.userRepo.GetByPhone(ctx, req.Phone)
		if err != nil {
			// Non-customer user doesn't exist - this should not be allowed for OTP
			// Non-customers should be created through regular registration
			return nil, errors.New("user not found. Please register first for non-customer roles")
		}

		// Verify role matches
		if user.Role != req.Role {
			return nil, errors.New("invalid role for this login")
		}
	}

	// Invalidate the OTP
	if err := s.otpRepo.InvalidateOTP(ctx, otp.ID); err != nil {
		// Log error but continue
		fmt.Printf("Failed to invalidate OTP: %v\n", err)
	}

	// Generate tokens
	restaurantIDStr := ""
	if user.RestaurantID != nil {
		restaurantIDStr = user.RestaurantID.String()
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID.String(), restaurantIDStr, user.Role, user.Email)
	if err != nil {
		return nil, err
	}

	// Store refresh token in Redis (30 days expiry)
	if err := s.storeRefreshToken(ctx, user.ID.String(), tokenPair.RefreshToken, 30); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
		User:         *user,
	}, nil
}

// Helper method to store refresh token (same as in AuthService)
func (s *OTPService) storeRefreshToken(ctx context.Context, userID, refreshToken string, expiryDays int) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	expiry := time.Hour * 24 * time.Duration(expiryDays)
	return s.cache.Set(ctx, key, refreshToken, expiry)
}

// CleanupExpiredOTPs removes expired OTPs from database
func (s *OTPService) CleanupExpiredOTPs(ctx context.Context) error {
	return s.otpRepo.DeleteExpiredOTPs(ctx)
}
