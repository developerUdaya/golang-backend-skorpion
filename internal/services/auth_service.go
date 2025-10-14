package services

import (
	"context"
	"errors"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/pkg/auth"
	"golang-food-backend/pkg/cache"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo   repositories.UserRepository
	jwtManager *auth.JWTManager
	cache      *cache.RedisCache
}

func NewAuthService(userRepo repositories.UserRepository, jwtManager *auth.JWTManager, cache *cache.RedisCache) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		cache:      cache,
	}
}

// Refresh token storage methods
func (s *AuthService) storeRefreshToken(ctx context.Context, userID, refreshToken string, expiryDays int) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	expiry := time.Hour * 24 * time.Duration(expiryDays)
	return s.cache.Set(ctx, key, refreshToken, expiry)
}

func (s *AuthService) getStoredRefreshToken(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", userID)
	var token string
	err := s.cache.Get(ctx, key, &token)
	return token, err
}

func (s *AuthService) invalidateRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return s.cache.Delete(ctx, key)
}

type RegisterRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone" binding:"required"`
	Password     string `json:"password" binding:"required,min=6"`
	Role         string `json:"role"`
	RestaurantID string `json:"restaurant_id" binding:"required"` // Required for multi-restaurant support
}

type LoginRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	RestaurantID string `json:"restaurant_id" binding:"required"` // Required for multi-restaurant support
}

type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"` // seconds until access token expires
	User         models.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Parse restaurant ID (required for multi-restaurant support)
	restaurantID, err := uuid.Parse(req.RestaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	// Check if user already exists within this restaurant
	existingUser, _ := s.userRepo.GetByEmailAndRestaurant(ctx, req.Email, restaurantID)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists in this restaurant")
	}

	existingUserByPhone, _ := s.userRepo.GetByPhoneAndRestaurant(ctx, req.Phone, restaurantID)
	if existingUserByPhone != nil {
		return nil, errors.New("user with this phone already exists in this restaurant")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Set default role
	role := req.Role
	if role == "" {
		role = "customer"
	}

	// Create user
	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hashedPassword),
		Role:         role,
		RestaurantID: &restaurantID,
		Status:       "active",
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate token pair
	restaurantIDStr := restaurantID.String()

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

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// Parse restaurant ID (required for multi-restaurant support)
	restaurantID, err := uuid.Parse(req.RestaurantID)
	if err != nil {
		return nil, errors.New("invalid restaurant ID")
	}

	// Get user by email within specific restaurant
	user, err := s.userRepo.GetByEmailAndRestaurant(ctx, req.Email, restaurantID)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate token pair
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

	// Cache user session
	sessionKey := "user_session:" + user.ID.String()
	s.cache.Set(ctx, sessionKey, user, time.Hour*24)

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
		User:         *user,
	}, nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	// Try cache first
	sessionKey := "user_session:" + userID
	var cachedUser models.User
	if err := s.cache.Get(ctx, sessionKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// Get from database
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	s.cache.Set(ctx, sessionKey, user, time.Hour*24)

	return user, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return err
	}

	// Update allowed fields
	if name, ok := updates["name"]; ok {
		if nameStr, ok := name.(string); ok {
			user.Name = nameStr
		}
	}

	if phone, ok := updates["phone"]; ok {
		if phoneStr, ok := phone.(string); ok {
			user.Phone = phoneStr
		}
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Clear cache
	sessionKey := "user_session:" + userID
	s.cache.Delete(ctx, sessionKey)

	return nil
}

// RefreshAccessToken validates refresh token and generates new access token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate the refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Ensure this is a refresh token
	if claims.TokenType != auth.RefreshToken {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	// Check if refresh token exists in storage
	storedToken, err := s.getStoredRefreshToken(ctx, claims.UserID)
	if err != nil || storedToken != refreshToken {
		return nil, errors.New("refresh token not found or invalid")
	}

	// Get user details
	userUUID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user, err := s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Keep the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
		User:         *user,
	}, nil
}

// Logout invalidates the refresh token
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	// Invalidate refresh token
	if err := s.invalidateRefreshToken(ctx, userID); err != nil {
		return err
	}

	// Remove user session from cache
	sessionKey := "user_session:" + userID
	return s.cache.Delete(ctx, sessionKey)
}
