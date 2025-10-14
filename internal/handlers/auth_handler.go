package handlers

import (
	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService AuthServiceInterface
	otpService  *services.OTPService
}

func NewAuthHandler(authService AuthServiceInterface, otpService *services.OTPService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		otpService:  otpService,
	}
}

// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.RegisterRequest true "Registration request"
// @Success 201 {object} services.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Login request"
// @Success 200 {object} services.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get user profile
// @Description Get current user's profile information
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.authService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Update user profile
// @Description Update current user's profile information
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Profile updates"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.UpdateProfile(c.Request.Context(), userID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// @Summary Refresh access token
// @Description Exchange refresh token for new access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token request"
// @Success 200 {object} services.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Logout user
// @Description Logout user and invalidate refresh token
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// @Summary Send OTP to phone
// @Description Send OTP for phone number authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.SendOTPRequest true "Send OTP request"
// @Success 200 {object} services.OTPResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/auth/send-otp [post]
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req services.SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.otpService.SendOTP(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Verify OTP and login
// @Description Verify OTP and authenticate user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.VerifyOTPRequest true "Verify OTP request"
// @Success 200 {object} services.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req services.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.otpService.VerifyOTPAndLogin(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)

		// OTP authentication endpoints
		auth.POST("/send-otp", h.SendOTP)
		auth.POST("/verify-otp", h.VerifyOTP)

		// Protected routes
		protected := auth.Group("/", authMiddleware.AuthRequired())
		{
			protected.GET("/profile", h.GetProfile)
			protected.PUT("/profile", h.UpdateProfile)
			protected.POST("/logout", h.Logout)
		}
	}
}
