package handlers

import (
	"context"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/services"
)

// AuthServiceInterface defines the interface for auth service operations
type AuthServiceInterface interface {
	Register(ctx context.Context, req *services.RegisterRequest) (*services.AuthResponse, error)
	Login(ctx context.Context, req *services.LoginRequest) (*services.AuthResponse, error)
	GetUserProfile(ctx context.Context, userID string) (*models.User, error)
	UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) error
	RefreshAccessToken(ctx context.Context, refreshToken string) (*services.AuthResponse, error)
	Logout(ctx context.Context, userID string) error
}
