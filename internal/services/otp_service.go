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
	RestaurantID string `json:"restaurant_id" binding:"required"`
}

type VerifyOTPRequest struct {
	Phone        string `json:"phone" binding:"required"`
	RestaurantID string `json:"restaurant_id" binding:"required"`
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
	// Parse restaurant ID
	restaurantID, err := uuid.Parse(req.RestaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	// Generate OTP
	otpCode, err := s.generateOTP()
	if err != nil {
		return nil, errors.New("failed to generate OTP")
	}

	// Create OTP record
	otp := &models.OTP{
		Phone:        req.Phone,
		RestaurantID: &restaurantID,
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
	// Parse restaurant ID
	restaurantID, err := uuid.Parse(req.RestaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	// Get valid OTP
	otp, err := s.otpRepo.GetValidOTP(ctx, req.Phone, restaurantID, req.OTPCode)
	if err != nil {
		return nil, errors.New("invalid or expired OTP")
	}

	// Check if user exists for this restaurant
	user, err := s.userRepo.GetByPhoneAndRestaurant(ctx, req.Phone, restaurantID)
	if err != nil {
		// User doesn't exist, create a new user
		user = &models.User{
			Name:         fmt.Sprintf("User-%s", req.Phone[len(req.Phone)-4:]), // Use last 4 digits
			Phone:        req.Phone,
			Email:        "", // Email can be empty for OTP login
			PasswordHash: "", // No password for OTP users
			RestaurantID: &restaurantID,
			Role:         "customer",
			Status:       "active",
			IsVerified:   true, // OTP verified means phone is verified
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, errors.New("failed to create user")
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
