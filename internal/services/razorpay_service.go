package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RazorpayService struct {
	apiKey          string
	apiSecret       string
	webhookSecret   string
	baseURL         string
	paymentRepo     repositories.PaymentRepository
	orderRepo       repositories.OrderRepository
	deliveryService *DeliveryPartnerService
}

func NewRazorpayService(
	apiKey, apiSecret, webhookSecret string,
	paymentRepo repositories.PaymentRepository,
	orderRepo repositories.OrderRepository,
	deliveryService *DeliveryPartnerService,
) *RazorpayService {
	return &RazorpayService{
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		webhookSecret:   webhookSecret,
		baseURL:         "https://api.razorpay.com/v1",
		paymentRepo:     paymentRepo,
		orderRepo:       orderRepo,
		deliveryService: deliveryService,
	}
}

type RazorpayOrderRequest struct {
	Amount         int                    `json:"amount"`   // Amount in paise
	Currency       string                 `json:"currency"` // INR
	Receipt        string                 `json:"receipt"`
	PartialPayment bool                   `json:"partial_payment"`
	Notes          map[string]interface{} `json:"notes,omitempty"`
}

type RazorpayOrderResponse struct {
	ID         string                 `json:"id"`
	Entity     string                 `json:"entity"`
	Amount     int                    `json:"amount"`
	AmountPaid int                    `json:"amount_paid"`
	AmountDue  int                    `json:"amount_due"`
	Currency   string                 `json:"currency"`
	Receipt    string                 `json:"receipt"`
	Status     string                 `json:"status"`
	Attempts   int                    `json:"attempts"`
	Notes      map[string]interface{} `json:"notes"`
	CreatedAt  int64                  `json:"created_at"`
}

type RazorpayWebhookPayload struct {
	Entity    string                 `json:"entity"`
	Account   string                 `json:"account_id"`
	Event     string                 `json:"event"`
	Contains  []string               `json:"contains"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt int64                  `json:"created_at"`
}

type PlaceOrderRequest struct {
	UserID          string                 `json:"user_id" binding:"required"`
	RestaurantID    string                 `json:"restaurant_id" binding:"required"`
	CartID          string                 `json:"cart_id" binding:"required"`
	AddressID       *string                `json:"address_id,omitempty"`
	Amount          float64                `json:"amount" binding:"required,min=1"`
	CustomerName    string                 `json:"customer_name" binding:"required"`
	CustomerContact string                 `json:"customer_contact" binding:"required"`
	DeliveryAddress map[string]interface{} `json:"delivery_address" binding:"required"`
}

type PlaceOrderResponse struct {
	OrderID         string `json:"order_id"`
	RazorpayOrderID string `json:"razorpay_order_id"`
	Amount          int    `json:"amount"` // in paise
	Currency        string `json:"currency"`
	PaymentID       string `json:"payment_id"`
}

// CreateRazorpayOrder creates a Razorpay order and stores payment record
func (s *RazorpayService) CreateRazorpayOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	// Create order in our database first
	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %v", err)
	}

	restaurantUUID, err := uuid.Parse(req.RestaurantID)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}

	cartUUID, err := uuid.Parse(req.CartID)
	if err != nil {
		return nil, fmt.Errorf("invalid cart ID: %v", err)
	}

	// Create order record
	order := &models.Order{
		ID:                             uuid.New(),
		UserID:                         userUUID,
		RestaurantID:                   restaurantUUID,
		CartID:                         cartUUID,
		OrderStatus:                    "pending_payment",
		TotalAmount:                    req.Amount,
		CustomerName:                   req.CustomerName,
		CustomerContact:                req.CustomerContact,
		DeliveryFullAddressWithLatLong: req.DeliveryAddress,
		CreatedAt:                      time.Now(),
	}

	if req.AddressID != nil {
		addressUUID, err := uuid.Parse(*req.AddressID)
		if err == nil {
			order.AddressID = &addressUUID
		}
	}

	// Save order to database
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Create Razorpay order
	amountInPaise := int(req.Amount * 100) // Convert to paise

	// Mock Razorpay order creation (in real implementation, make HTTP request to Razorpay)
	razorpayOrderID := fmt.Sprintf("order_%s", uuid.New().String()[:8])

	// Create payment record
	payment := &models.Payment{
		ID:            uuid.New(),
		OrderID:       order.ID,
		UserID:        userUUID,
		Amount:        req.Amount,
		Method:        "razorpay",
		Status:        "pending",
		TransactionID: razorpayOrderID,
		CreatedAt:     time.Now(),
		Metadata: models.JSONB{
			"razorpay_order_id": razorpayOrderID,
			"currency":          "INR",
			"amount_paise":      amountInPaise,
		},
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %v", err)
	}

	// Update order with payment ID
	order.PaymentID = &payment.ID
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order with payment ID: %v", err)
	}

	return &PlaceOrderResponse{
		OrderID:         order.ID.String(),
		RazorpayOrderID: razorpayOrderID,
		Amount:          amountInPaise,
		Currency:        "INR",
		PaymentID:       payment.ID.String(),
	}, nil
}

// HandlePaymentWebhook processes Razorpay webhooks for payment updates
func (s *RazorpayService) HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error {
	// Verify webhook signature
	if !s.verifyWebhookSignature(payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	var webhook RazorpayWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %v", err)
	}

	// Handle payment success/failure events
	switch webhook.Event {
	case "payment.captured", "payment.authorized":
		return s.handlePaymentSuccess(ctx, webhook.Payload)
	case "payment.failed":
		return s.handlePaymentFailure(ctx, webhook.Payload)
	default:
		// Log unhandled event but don't return error
		fmt.Printf("Unhandled webhook event: %s\n", webhook.Event)
		return nil
	}
}

func (s *RazorpayService) handlePaymentSuccess(ctx context.Context, payload map[string]interface{}) error {
	paymentData, ok := payload["payment"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payment data in webhook")
	}

	razorpayOrderID, ok := paymentData["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing order_id in payment data")
	}

	// Find payment by razorpay order ID
	payment, err := s.paymentRepo.GetByTransactionID(ctx, razorpayOrderID)
	if err != nil {
		return fmt.Errorf("payment not found for order ID %s: %v", razorpayOrderID, err)
	}

	// Update payment status
	payment.Status = "success"
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Update order status
	order, err := s.orderRepo.GetByID(ctx, payment.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %v", err)
	}

	order.OrderStatus = "confirmed"
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update order status: %v", err)
	}

	// Create delivery order with restaurant's delivery partner
	if s.deliveryService != nil {
		go func() {
			if err := s.deliveryService.CreateDeliveryOrder(context.Background(), order); err != nil {
				fmt.Printf("Failed to create delivery order for order %s: %v\n", order.ID.String(), err)
			}
		}()
	}

	return nil
}

func (s *RazorpayService) handlePaymentFailure(ctx context.Context, payload map[string]interface{}) error {
	paymentData, ok := payload["payment"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payment data in webhook")
	}

	razorpayOrderID, ok := paymentData["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing order_id in payment data")
	}

	// Find payment by razorpay order ID
	payment, err := s.paymentRepo.GetByTransactionID(ctx, razorpayOrderID)
	if err != nil {
		return fmt.Errorf("payment not found for order ID %s: %v", razorpayOrderID, err)
	}

	// Update payment status
	payment.Status = "failed"
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Delete the order as payment failed
	if err := s.orderRepo.Delete(ctx, payment.OrderID); err != nil {
		return fmt.Errorf("failed to delete order after payment failure: %v", err)
	}

	return nil
}

// verifyWebhookSignature verifies the Razorpay webhook signature
func (s *RazorpayService) verifyWebhookSignature(payload []byte, signature string) bool {
	expectedSignature := s.generateWebhookSignature(payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (s *RazorpayService) generateWebhookSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(s.webhookSecret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Mock method for real Razorpay API call (replace with actual HTTP client call)
func (s *RazorpayService) createRazorpayOrderAPI(req *RazorpayOrderRequest) (*RazorpayOrderResponse, error) {
	// This would be a real HTTP request to Razorpay API
	// For now, returning a mock response

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", s.baseURL+"/orders", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.SetBasicAuth(s.apiKey, s.apiSecret)
	httpReq.Header.Set("Content-Type", "application/json")

	// Mock response for now
	mockResponse := &RazorpayOrderResponse{
		ID:        fmt.Sprintf("order_%s", uuid.New().String()[:8]),
		Entity:    "order",
		Amount:    req.Amount,
		Currency:  req.Currency,
		Receipt:   req.Receipt,
		Status:    "created",
		CreatedAt: time.Now().Unix(),
		Notes:     req.Notes,
	}

	return mockResponse, nil
}
