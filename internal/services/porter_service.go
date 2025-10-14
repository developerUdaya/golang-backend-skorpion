package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"io"
	"net/http"
	"os"
	"time"
)

type PorterService struct {
	apiKey             string
	baseURL            string
	httpClient         *http.Client
	orderRepo          repositories.OrderRepository
	porterDeliveryRepo repositories.PorterDeliveryRepository
}

func NewPorterService(orderRepo repositories.OrderRepository, porterDeliveryRepo repositories.PorterDeliveryRepository) *PorterService {
	apiKey := os.Getenv("PORTER_API_KEY")
	baseURL := os.Getenv("PORTER_BASE_URL")

	if apiKey == "" {
		apiKey = "O8AJTXXXXXXXXXX-UA1LiA" // Default test key
	}
	if baseURL == "" {
		baseURL = "https://pfe-apigw-uat.porter.in" // Default test URL
	}

	return &PorterService{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		orderRepo:          orderRepo,
		porterDeliveryRepo: porterDeliveryRepo,
	}
}

// Porter API Request/Response Structures

// GetQuote structures
type PorterQuoteRequest struct {
	PickupDetails struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"pickup_details"`
	DropDetails struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"drop_details"`
	Customer struct {
		Name   string `json:"name"`
		Mobile struct {
			CountryCode string `json:"country_code"`
			Number      string `json:"number"`
		} `json:"mobile"`
	} `json:"customer"`
}

type PorterQuoteResponse struct {
	EstimatedFare struct {
		Currency    string `json:"currency"`
		MinorAmount int    `json:"minor_amount"`
	} `json:"estimated_fare"`
	EstimatedTime int    `json:"estimated_time"`
	VehicleType   string `json:"vehicle_type"`
	Distance      string `json:"distance"`
}

// Create Order structures
type PorterCreateOrderRequest struct {
	RequestID            string                      `json:"request_id"`
	DeliveryInstructions *PorterDeliveryInstructions `json:"delivery_instructions,omitempty"`
	PickupDetails        PorterAddressDetails        `json:"pickup_details"`
	DropDetails          PorterAddressDetails        `json:"drop_details"`
	AdditionalComments   string                      `json:"additional_comments,omitempty"`
}

type PorterDeliveryInstructions struct {
	InstructionsList []PorterInstruction `json:"instructions_list"`
}

type PorterInstruction struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type PorterAddressDetails struct {
	Address PorterAddress `json:"address"`
}

type PorterAddress struct {
	ApartmentAddress string               `json:"apartment_address"`
	StreetAddress1   string               `json:"street_address1"`
	StreetAddress2   string               `json:"street_address2"`
	Landmark         string               `json:"landmark"`
	City             string               `json:"city"`
	State            string               `json:"state"`
	Pincode          string               `json:"pincode"`
	Country          string               `json:"country"`
	Lat              float64              `json:"lat"`
	Lng              float64              `json:"lng"`
	ContactDetails   PorterContactDetails `json:"contact_details"`
}

type PorterContactDetails struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
}

type PorterCreateOrderResponse struct {
	RequestID            string            `json:"request_id"`
	OrderID              string            `json:"order_id"`
	EstimatedPickupTime  int64             `json:"estimated_pickup_time"`
	EstimatedFareDetails PorterFareDetails `json:"estimated_fare_details"`
	TrackingURL          string            `json:"tracking_url"`
}

type PorterFareDetails struct {
	Currency    string `json:"currency"`
	MinorAmount int    `json:"minor_amount"`
}

// Track Order structures
type PorterTrackOrderResponse struct {
	OrderID      string                 `json:"order_id"`
	Status       string                 `json:"status"`
	PartnerInfo  *PorterPartnerInfo     `json:"partner_info,omitempty"`
	OrderTimings PorterOrderTimings     `json:"order_timings"`
	FareDetails  PorterTrackFareDetails `json:"fare_details"`
}

type PorterPartnerInfo struct {
	Name          string `json:"name"`
	VehicleNumber string `json:"vehicle_number"`
	VehicleType   string `json:"vehicle_type"`
	Mobile        struct {
		CountryCode  string `json:"country_code"`
		MobileNumber string `json:"mobile_number"`
	} `json:"mobile"`
	PartnerSecondaryMobile struct {
		CountryCode  string `json:"country_code"`
		MobileNumber string `json:"mobile_number"`
	} `json:"partner_secondary_mobile"`
	Location *struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"long"`
	} `json:"location,omitempty"`
}

type PorterOrderTimings struct {
	PickupTime        *int64 `json:"pickup_time"`
	OrderAcceptedTime *int64 `json:"order_accepted_time"`
	OrderStartedTime  *int64 `json:"order_started_time"`
	OrderEndedTime    *int64 `json:"order_ended_time"`
}

type PorterTrackFareDetails struct {
	EstimatedFareDetails *PorterFareDetails `json:"estimated_fare_details"`
	ActualFareDetails    *PorterFareDetails `json:"actual_fare_details"`
}

// Cancel Order structures
type PorterCancelOrderResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Webhook structures
type PorterWebhookPayload struct {
	Status       string                    `json:"status"`
	OrderID      string                    `json:"order_id"`
	OrderDetails PorterWebhookOrderDetails `json:"order_details"`
}

type PorterWebhookOrderDetails struct {
	EventTs           int64                `json:"event_ts"`
	PartnerLocation   *PorterLocation      `json:"partner_location,omitempty"`
	DriverDetails     *PorterDriverDetails `json:"driver_details,omitempty"`
	EstimatedTripFare *float64             `json:"estimated_trip_fare,omitempty"`
	ActualTripFare    *float64             `json:"actual_trip_fare,omitempty"`
}

type PorterLocation struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

type PorterDriverDetails struct {
	DriverName    string `json:"driver_name"`
	VehicleNumber string `json:"vehicle_number"`
	Mobile        string `json:"mobile"`
}

// Service Methods

// GetQuote gets delivery quote from Porter
func (s *PorterService) GetQuote(ctx context.Context, req *PorterQuoteRequest) (*PorterQuoteResponse, error) {
	url := fmt.Sprintf("%s/v1/get_quote", s.baseURL)

	respBody, err := s.makePorterRequest(ctx, "GET", url, req)
	if err != nil {
		return nil, err
	}

	var result PorterQuoteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quote response: %v", err)
	}

	return &result, nil
}

// CreateOrder creates a delivery order with Porter
func (s *PorterService) CreateOrder(ctx context.Context, req *PorterCreateOrderRequest) (*PorterCreateOrderResponse, error) {
	url := fmt.Sprintf("%s/v1/orders/create", s.baseURL)

	respBody, err := s.makePorterRequest(ctx, "POST", url, req)
	if err != nil {
		return nil, err
	}

	var result PorterCreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create order response: %v", err)
	}

	return &result, nil
}

// TrackOrder tracks a Porter order
func (s *PorterService) TrackOrder(ctx context.Context, orderID string) (*PorterTrackOrderResponse, error) {
	url := fmt.Sprintf("%s/v1/orders/%s", s.baseURL, orderID)

	respBody, err := s.makePorterRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result PorterTrackOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track order response: %v", err)
	}

	return &result, nil
}

// CancelOrder cancels a Porter order
func (s *PorterService) CancelOrder(ctx context.Context, orderID string) (*PorterCancelOrderResponse, error) {
	url := fmt.Sprintf("%s/v1/orders/%s/cancel", s.baseURL, orderID)

	respBody, err := s.makePorterRequest(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	var result PorterCancelOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cancel order response: %v", err)
	}

	return &result, nil
}

// Helper method to make Porter API requests
func (s *PorterService) makePorterRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set Porter API headers
	req.Header.Set("X-API-KEY", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Porter API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Business logic methods

// CreateDeliveryOrder creates a Porter delivery order for a food order
func (s *PorterService) CreateDeliveryOrder(ctx context.Context, order *models.Order, restaurant *models.Restaurant) (*PorterCreateOrderResponse, error) {
	// First get a quote
	quoteReq := &PorterQuoteRequest{}

	// Set pickup details (restaurant) - using mock coordinates for now
	quoteReq.PickupDetails.Lat = 12.935025018880504
	quoteReq.PickupDetails.Lng = 77.6092605236106

	// Set drop details (customer)
	quoteReq.DropDetails.Lat = s.extractLatitude(order.DeliveryFullAddressWithLatLong)
	quoteReq.DropDetails.Lng = s.extractLongitude(order.DeliveryFullAddressWithLatLong)

	// Set customer details
	quoteReq.Customer.Name = order.CustomerName
	quoteReq.Customer.Mobile.CountryCode = "+91"
	quoteReq.Customer.Mobile.Number = s.extractPhoneNumber(order.CustomerContact)

	// Get quote first
	quote, err := s.GetQuote(ctx, quoteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get Porter quote: %v", err)
	}

	// Create order request
	createReq := &PorterCreateOrderRequest{
		RequestID: fmt.Sprintf("FOOD_%s_%d", order.ID.String()[:8], time.Now().Unix()),
		DeliveryInstructions: &PorterDeliveryInstructions{
			InstructionsList: []PorterInstruction{
				{
					Type:        "text",
					Description: "Handle with care - Food delivery",
				},
			},
		},
		PickupDetails: PorterAddressDetails{
			Address: PorterAddress{
				ApartmentAddress: restaurant.Name,
				StreetAddress1:   "Restaurant Location", // Mock address
				StreetAddress2:   fmt.Sprintf("Restaurant - %s", restaurant.Name),
				Landmark:         "Restaurant Area",
				City:             "Bengaluru",
				State:            "Karnataka",
				Pincode:          "560029",
				Country:          "India",
				Lat:              12.935025018880504,
				Lng:              77.6092605236106,
				ContactDetails: PorterContactDetails{
					Name:        restaurant.Name,
					PhoneNumber: restaurant.ContactNumber,
				},
			},
		},
		DropDetails: PorterAddressDetails{
			Address: PorterAddress{
				ApartmentAddress: s.extractApartmentAddress(order.DeliveryFullAddressWithLatLong),
				StreetAddress1:   s.extractStreetAddress1(order.DeliveryFullAddressWithLatLong),
				StreetAddress2:   fmt.Sprintf("Order ID: %s", order.ID.String()),
				Landmark:         s.extractLandmark(order.DeliveryFullAddressWithLatLong),
				City:             s.extractCity(order.DeliveryFullAddressWithLatLong),
				State:            s.extractState(order.DeliveryFullAddressWithLatLong),
				Pincode:          s.extractPincode(order.DeliveryFullAddressWithLatLong),
				Country:          "India",
				Lat:              s.extractLatitude(order.DeliveryFullAddressWithLatLong),
				Lng:              s.extractLongitude(order.DeliveryFullAddressWithLatLong),
				ContactDetails: PorterContactDetails{
					Name:        order.CustomerName,
					PhoneNumber: order.CustomerContact,
				},
			},
		},
		AdditionalComments: fmt.Sprintf("Food delivery from %s. Order value: ₹%.2f", restaurant.Name, order.TotalAmount),
	}

	// Create the order
	porterOrder, err := s.CreateOrder(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Porter order: %v", err)
	}

	// Update our order with Porter details
	if order.OrderLogs == nil {
		order.OrderLogs = make(models.JSONB)
	}

	orderLog := map[string]interface{}{
		"action":           "porter_order_created",
		"porter_order_id":  porterOrder.OrderID,
		"tracking_url":     porterOrder.TrackingURL,
		"estimated_fare":   float64(porterOrder.EstimatedFareDetails.MinorAmount) / 100, // Convert from minor units
		"estimated_pickup": porterOrder.EstimatedPickupTime,
		"quote_distance":   quote.Distance,
		"quote_vehicle":    quote.VehicleType,
		"timestamp":        time.Now().Unix(),
	}

	// Add to existing order logs
	logs, ok := order.OrderLogs["logs"].([]interface{})
	if !ok {
		logs = []interface{}{}
	}
	logs = append(logs, orderLog)
	order.OrderLogs["logs"] = logs

	// Save order updates
	if err := s.orderRepo.Update(ctx, order); err != nil {
		// Log error but don't fail the order creation
		fmt.Printf("Failed to update order with Porter details: %v\n", err)
	}

	return porterOrder, nil
}

// HandleWebhook processes Porter webhook notifications
func (s *PorterService) HandleWebhook(ctx context.Context, payload *PorterWebhookPayload) error {
	// Find Porter delivery by Porter order ID
	porterDelivery, err := s.porterDeliveryRepo.GetByPorterOrderID(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("failed to find Porter delivery with order ID %s: %w", payload.OrderID, err)
	}

	// Update Porter delivery status and details
	porterDelivery.Status = payload.Status
	porterDelivery.UpdatedAt = time.Now()

	// Handle different webhook statuses
	switch payload.Status {
	case "order_accepted":
		// Update partner details when order is accepted
		porterDelivery.PartnerName = payload.OrderDetails.DriverDetails.DriverName
		porterDelivery.PartnerPhoneNumber = payload.OrderDetails.DriverDetails.Mobile
		porterDelivery.VehicleNumber = payload.OrderDetails.DriverDetails.VehicleNumber

		// Update estimated delivery time if provided
		if payload.OrderDetails.EventTs > 0 {
			estimatedTime := time.Unix(payload.OrderDetails.EventTs, 0)
			porterDelivery.EstimatedDeliveryTime = &estimatedTime
		}

	case "order_start_trip":
		// Update pickup time when trip starts
		pickupTime := time.Now()
		porterDelivery.PickupTime = &pickupTime

	case "order_end_job":
		// Update delivery completion time when job ends
		deliveryTime := time.Now()
		porterDelivery.ActualDeliveryTime = &deliveryTime

	case "order_reopen":
		// Reset delivery time if order is reopened
		porterDelivery.ActualDeliveryTime = nil

	case "order_cancel":
		// Mark as inactive when cancelled
		porterDelivery.IsActive = false
	}

	// Save updated Porter delivery
	if err := s.porterDeliveryRepo.Update(ctx, porterDelivery); err != nil {
		return fmt.Errorf("failed to update Porter delivery: %w", err)
	}

	// Update main order status based on Porter status
	order, err := s.orderRepo.GetByID(ctx, porterDelivery.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Map Porter status to order status
	var newOrderStatus string
	switch payload.Status {
	case "order_accepted":
		newOrderStatus = "dispatched" // Delivery partner assigned and heading to pickup
	case "order_start_trip":
		newOrderStatus = "dispatched" // Package picked up, heading to customer
	case "order_end_job":
		newOrderStatus = "delivered" // Successfully delivered
	case "order_cancel":
		newOrderStatus = "cancelled" // Delivery cancelled
	case "order_reopen":
		newOrderStatus = "dispatched" // Back in transit
	}

	if newOrderStatus != "" && order.OrderStatus != newOrderStatus {
		order.OrderStatus = newOrderStatus

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}
	}

	return nil
}

// Helper methods for address extraction

func (s *PorterService) extractLatitude(addressData models.JSONB) float64 {
	if addressData == nil {
		return 12.947146336879577
	}
	if lat, ok := addressData["latitude"].(float64); ok {
		return lat
	}
	return 12.947146336879577
}

func (s *PorterService) extractLongitude(addressData models.JSONB) float64 {
	if addressData == nil {
		return 77.62102993895199
	}
	if lng, ok := addressData["longitude"].(float64); ok {
		return lng
	}
	return 77.62102993895199
}

func (s *PorterService) extractPhoneNumber(contact string) string {
	// Remove country code if present
	if len(contact) > 10 && contact[:3] == "+91" {
		return contact[3:]
	}
	if len(contact) > 10 && contact[:2] == "91" {
		return contact[2:]
	}
	return contact
}

func (s *PorterService) extractApartmentAddress(addressData models.JSONB) string {
	if addressData == nil {
		return ""
	}
	if apt, ok := addressData["apartment"].(string); ok {
		return apt
	}
	return ""
}

func (s *PorterService) extractStreetAddress1(addressData models.JSONB) string {
	if addressData == nil {
		return "Street Address"
	}
	if addr, ok := addressData["line1"].(string); ok {
		return addr
	}
	return "Street Address"
}

func (s *PorterService) extractLandmark(addressData interface{}) string {
	if data, ok := addressData.(models.JSONB); ok {
		if landmark, ok := data["landmark"].(string); ok {
			return landmark
		}
	}
	return ""
}

func (s *PorterService) extractCity(addressData interface{}) string {
	if data, ok := addressData.(models.JSONB); ok {
		if city, ok := data["city"].(string); ok {
			return city
		}
	}
	return "Bengaluru"
}

func (s *PorterService) extractState(addressData interface{}) string {
	if data, ok := addressData.(models.JSONB); ok {
		if state, ok := data["state"].(string); ok {
			return state
		}
	}
	return "Karnataka"
}

func (s *PorterService) extractPincode(addressData interface{}) string {
	if data, ok := addressData.(models.JSONB); ok {
		if pincode, ok := data["pincode"].(string); ok {
			return pincode
		}
	}
	return "560029"
}
