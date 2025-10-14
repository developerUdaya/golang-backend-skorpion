package handlers

import (
	"context"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PorterHandler struct {
	porterService          *services.PorterService
	porterDeliveryRepo     repositories.PorterDeliveryRepository
	orderRepo              repositories.OrderRepository
	deliveryPartnerService *services.DeliveryPartnerService
}

func NewPorterHandler(
	porterService *services.PorterService,
	porterDeliveryRepo repositories.PorterDeliveryRepository,
	orderRepo repositories.OrderRepository,
	deliveryPartnerService *services.DeliveryPartnerService,
) *PorterHandler {
	return &PorterHandler{
		porterService:          porterService,
		porterDeliveryRepo:     porterDeliveryRepo,
		orderRepo:              orderRepo,
		deliveryPartnerService: deliveryPartnerService,
	}
}

// GetQuote handles getting delivery quote from Porter
func (h *PorterHandler) GetQuote(c *gin.Context) {
	var req services.PorterQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Validate required fields
	if req.PickupDetails.Lat == 0 || req.PickupDetails.Lng == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pickup location coordinates are required"})
		return
	}

	if req.DropDetails.Lat == 0 || req.DropDetails.Lng == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Drop location coordinates are required"})
		return
	}

	if req.Customer.Name == "" || req.Customer.Mobile.Number == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Customer name and mobile number are required"})
		return
	}

	quote, err := h.porterService.GetQuote(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get quote: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, quote)
}

// CreateDeliveryOrder handles creating a delivery order with Porter
func (h *PorterHandler) CreateDeliveryOrder(c *gin.Context) {
	var req services.PorterCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Validate required fields
	if req.PickupDetails.Address.ContactDetails.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pickup contact name is required"})
		return
	}

	if req.DropDetails.Address.ContactDetails.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Drop contact name is required"})
		return
	}

	order, err := h.porterService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery order: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// TrackOrder handles tracking a Porter delivery order
func (h *PorterHandler) TrackOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	trackingInfo, err := h.porterService.TrackOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track order: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, trackingInfo)
}

// CancelOrder handles cancelling a Porter delivery order
func (h *PorterHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	result, err := h.porterService.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Webhook handles Porter webhook notifications
func (h *PorterHandler) Webhook(c *gin.Context) {
	var payload services.PorterWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload: " + err.Error()})
		return
	}

	// Validate webhook payload
	if payload.OrderID == "" || payload.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID and status are required in webhook payload"})
		return
	}

	// Find Porter delivery by Porter order ID
	porterDelivery, err := h.porterDeliveryRepo.GetByPorterOrderID(c.Request.Context(), payload.OrderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":           "Porter delivery not found for order ID: " + payload.OrderID,
			"porter_order_id": payload.OrderID,
			"status":          payload.Status,
		})
		return
	}

	// Get the associated food order
	order, err := h.orderRepo.GetByID(c.Request.Context(), porterDelivery.OrderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":           "Order not found for Porter delivery",
			"porter_order_id": payload.OrderID,
			"order_id":        porterDelivery.OrderID,
		})
		return
	}

	// Store original order status for comparison
	originalOrderStatus := order.OrderStatus

	// Handle Porter webhook through service (this updates Porter delivery and order status)
	err = h.porterService.HandleWebhook(c.Request.Context(), &payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":           "Failed to process webhook: " + err.Error(),
			"porter_order_id": payload.OrderID,
			"order_id":        order.ID,
		})
		return
	}

	// Handle voluntary cancellation by Porter - trigger reassignment
	if payload.Status == "order_cancel" {
		// Check if this was a voluntary cancellation by Porter (not customer-initiated)
		// We can identify this by checking if the order status was not already "cancelled"
		if originalOrderStatus != "cancelled" {
			// Porter voluntarily cancelled - attempt reassignment
			go func() {
				// Create a background context with timeout for reassignment
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Run reassignment in background to avoid blocking webhook response
				reassignErr := h.deliveryPartnerService.ReassignDeliveryPartner(ctx, order.ID, "porter")
				if reassignErr != nil {
					// Log the error - in production, you'd use a proper logger
					// logger.Error("Failed to reassign Porter delivery after voluntary cancellation",
					//   "order_id", order.ID, "porter_order_id", payload.OrderID, "error", reassignErr)
				}
			}()
		}
	}

	// Return success response with detailed information
	c.JSON(http.StatusOK, gin.H{
		"message":                "Webhook processed successfully",
		"porter_status":          payload.Status,
		"order_id":               order.ID,
		"porter_order_id":        payload.OrderID,
		"order_status":           order.OrderStatus,
		"reassignment_triggered": payload.Status == "order_cancel" && originalOrderStatus != "cancelled",
	})
}

// Helper endpoint to create Porter delivery order for existing food order
func (h *PorterHandler) CreateDeliveryOrderForFoodOrder(c *gin.Context) {
	orderIDStr := c.Param("order_id")
	if orderIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	// This would require order and restaurant repositories to be injected
	// For now, return a placeholder response
	c.JSON(http.StatusNotImplemented, gin.H{
		"message":  "This endpoint requires order and restaurant repositories to be implemented",
		"order_id": orderID,
	})
}

// ReassignPorterDelivery handles reassigning Porter delivery for an existing order
func (h *PorterHandler) ReassignPorterDelivery(c *gin.Context) {
	orderIDStr := c.Param("order_id")
	if orderIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	// Get the existing order
	_, err = h.orderRepo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Deactivate old Porter deliveries for this order
	if err := h.porterDeliveryRepo.DeactivateOldDeliveries(c.Request.Context(), orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate old deliveries"})
		return
	}

	// Cancel existing Porter orders if any
	activeDelivery, err := h.porterDeliveryRepo.GetActiveByOrderID(c.Request.Context(), orderID)
	if err == nil && activeDelivery != nil {
		// Cancel the old Porter order
		_, cancelErr := h.porterService.CancelOrder(c.Request.Context(), activeDelivery.PorterOrderID)
		if cancelErr != nil {
			// Log the error but continue with reassignment
			// logger.Warn("Failed to cancel old Porter order", "error", cancelErr)
		}
	}

	// Create new delivery assignment using the delivery partner service
	err = h.deliveryPartnerService.ReassignDeliveryPartner(c.Request.Context(), orderID, "porter")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reassign Porter delivery: " + err.Error()})
		return
	}

	// Get the updated order with new Porter delivery details
	updatedOrder, err := h.orderRepo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Porter delivery reassigned successfully",
		"order":   updatedOrder,
	})
}

// RegisterRoutes registers all Porter-related routes
func (h *PorterHandler) RegisterRoutes(router *gin.RouterGroup) {
	porter := router.Group("/porter")
	{
		// Quote management
		porter.POST("/quote", h.GetQuote)

		// Order management
		porter.POST("/orders", h.CreateDeliveryOrder)
		porter.GET("/orders/:order_id", h.TrackOrder)
		porter.POST("/orders/:order_id/cancel", h.CancelOrder)

		// Webhook
		porter.POST("/webhook", h.Webhook)

		// Helper endpoints
		porter.POST("/create-delivery/:order_id", h.CreateDeliveryOrderForFoodOrder)

		// Reassignment endpoint
		porter.POST("/reassign/:order_id", h.ReassignPorterDelivery)
	}
}

// Sample request/response structures for documentation

type GetQuoteRequest struct {
	PickupDetails struct {
		Lat float64 `json:"lat" example:"12.935025018880504"`
		Lng float64 `json:"lng" example:"77.6092605236106"`
	} `json:"pickup_details"`
	DropDetails struct {
		Lat float64 `json:"lat" example:"12.947146336879577"`
		Lng float64 `json:"lng" example:"77.62102993895199"`
	} `json:"drop_details"`
	Customer struct {
		Name   string `json:"name" example:"John Doe"`
		Mobile struct {
			CountryCode string `json:"country_code" example:"+91"`
			Number      string `json:"number" example:"9876543210"`
		} `json:"mobile"`
	} `json:"customer"`
}

type CreateOrderRequest struct {
	RequestID            string                `json:"request_id" example:"FOOD_12345_1640995200"`
	DeliveryInstructions *DeliveryInstructions `json:"delivery_instructions,omitempty"`
	PickupDetails        AddressDetails        `json:"pickup_details"`
	DropDetails          AddressDetails        `json:"drop_details"`
	AdditionalComments   string                `json:"additional_comments,omitempty" example:"Handle with care - Food delivery"`
}

type DeliveryInstructions struct {
	InstructionsList []Instruction `json:"instructions_list"`
}

type Instruction struct {
	Type        string `json:"type" example:"text"`
	Description string `json:"description" example:"Handle with care"`
}

type AddressDetails struct {
	Address Address `json:"address"`
}

type Address struct {
	ApartmentAddress string         `json:"apartment_address" example:"Apartment 27"`
	StreetAddress1   string         `json:"street_address1" example:"Sona Towers"`
	StreetAddress2   string         `json:"street_address2" example:"Krishna Nagar Industrial Area"`
	Landmark         string         `json:"landmark" example:"Hosur Road"`
	City             string         `json:"city" example:"Bengaluru"`
	State            string         `json:"state" example:"Karnataka"`
	Pincode          string         `json:"pincode" example:"560029"`
	Country          string         `json:"country" example:"India"`
	Lat              float64        `json:"lat" example:"12.935025018880504"`
	Lng              float64        `json:"lng" example:"77.6092605236106"`
	ContactDetails   ContactDetails `json:"contact_details"`
}

type ContactDetails struct {
	Name        string `json:"name" example:"John Doe"`
	PhoneNumber string `json:"phone_number" example:"+919876543210"`
}

// Example responses for documentation
type QuoteResponse struct {
	EstimatedFare struct {
		Currency    string `json:"currency" example:"INR"`
		MinorAmount int    `json:"minor_amount" example:"3500"`
	} `json:"estimated_fare"`
	EstimatedTime int    `json:"estimated_time" example:"25"`
	VehicleType   string `json:"vehicle_type" example:"TWO_WHEELER"`
	Distance      string `json:"distance" example:"5.2 km"`
}

type CreateOrderResponse struct {
	RequestID            string `json:"request_id" example:"FOOD_12345_1640995200"`
	OrderID              string `json:"order_id" example:"CRN17855725"`
	EstimatedPickupTime  int64  `json:"estimated_pickup_time" example:"1642473111"`
	EstimatedFareDetails struct {
		Currency    string `json:"currency" example:"INR"`
		MinorAmount int    `json:"minor_amount" example:"3500"`
	} `json:"estimated_fare_details"`
	TrackingURL string `json:"tracking_url" example:"https://porter.in/track_live_order?booking_id=CRN83543479"`
}

type TrackOrderResponse struct {
	OrderID     string `json:"order_id" example:"CRN93814651"`
	Status      string `json:"status" example:"live"`
	PartnerInfo struct {
		Name          string `json:"name" example:"Anupam Patel"`
		VehicleNumber string `json:"vehicle_number" example:"AK-02-HH-2020"`
		VehicleType   string `json:"vehicle_type" example:"TWO_WHEELER"`
		Mobile        struct {
			CountryCode  string `json:"country_code" example:"91"`
			MobileNumber string `json:"mobile_number" example:"9535321734"`
		} `json:"mobile"`
		Location struct {
			Lat  float64 `json:"lat" example:"12.934672"`
			Long float64 `json:"long" example:"77.6093797"`
		} `json:"location"`
	} `json:"partner_info"`
	OrderTimings struct {
		PickupTime        int64  `json:"pickup_time" example:"1669879581"`
		OrderAcceptedTime int64  `json:"order_accepted_time" example:"1669877932"`
		OrderStartedTime  int64  `json:"order_started_time" example:"1669877997"`
		OrderEndedTime    *int64 `json:"order_ended_time"`
	} `json:"order_timings"`
	FareDetails struct {
		EstimatedFareDetails struct {
			Currency    string `json:"currency" example:"INR"`
			MinorAmount int    `json:"minor_amount" example:"8400"`
		} `json:"estimated_fare_details"`
		ActualFareDetails *struct {
			Currency    string `json:"currency" example:"INR"`
			MinorAmount int    `json:"minor_amount" example:"5500"`
		} `json:"actual_fare_details"`
	} `json:"fare_details"`
}

type WebhookPayload struct {
	Status       string `json:"status" example:"order_accepted"`
	OrderID      string `json:"order_id" example:"CRN123456789"`
	OrderDetails struct {
		EventTs         int64 `json:"event_ts" example:"1664457558"`
		PartnerLocation struct {
			Lat  float64 `json:"lat" example:"12.93468735"`
			Long float64 `json:"long" example:"77.6095961"`
		} `json:"partner_location,omitempty"`
		DriverDetails struct {
			DriverName    string `json:"driver_name" example:"Test Partner"`
			VehicleNumber string `json:"vehicle_number" example:"HY-45-YU-5677"`
			Mobile        string `json:"mobile" example:"6100001111"`
		} `json:"driver_details,omitempty"`
		EstimatedTripFare *float64 `json:"estimated_trip_fare,omitempty" example:"77"`
		ActualTripFare    *float64 `json:"actual_trip_fare,omitempty" example:"77"`
	} `json:"order_details"`
}
