package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type JWTManager struct {
	secretKey         string
	accessExpiryHours int
	refreshExpiryDays int
}

type Claims struct {
	UserID       string    `json:"user_id"`
	RestaurantID string    `json:"restaurant_id"`
	Role         string    `json:"role"`
	Email        string    `json:"email"`
	TokenType    TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewJWTManager(secretKey string, accessExpiryHours, refreshExpiryDays int) *JWTManager {
	return &JWTManager{
		secretKey:         secretKey,
		accessExpiryHours: accessExpiryHours,
		refreshExpiryDays: refreshExpiryDays,
	}
}

func (j *JWTManager) generateToken(userID, restaurantID, role, email string, tokenType TokenType) (string, error) {
	var expiryTime time.Time
	if tokenType == AccessToken {
		expiryTime = time.Now().Add(time.Hour * time.Duration(j.accessExpiryHours))
	} else {
		expiryTime = time.Now().Add(time.Hour * 24 * time.Duration(j.refreshExpiryDays))
	}

	claims := &Claims{
		UserID:       userID,
		RestaurantID: restaurantID,
		Role:         role,
		Email:        email,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiryTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTManager) GenerateToken(userID, restaurantID, role, email string) (string, error) {
	return j.generateToken(userID, restaurantID, role, email, AccessToken)
}

func (j *JWTManager) GenerateTokenPair(userID, restaurantID, role, email string) (*TokenPair, error) {
	accessToken, err := j.generateToken(userID, restaurantID, role, email, AccessToken)
	if err != nil {
		return nil, err
	}

	refreshToken, err := j.generateToken(userID, restaurantID, role, email, RefreshToken)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTManager) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Ensure this is a refresh token
	if claims.TokenType != RefreshToken {
		return "", errors.New("invalid token type: expected refresh token")
	}

	// Generate new access token
	return j.generateToken(claims.UserID, claims.RestaurantID, claims.Role, claims.Email, AccessToken)
}
