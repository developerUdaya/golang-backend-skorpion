package sms

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type SMSService struct {
	apiKey   string
	senderID string
	baseURL  string
}

type SMSResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewSMSService(apiKey, senderID string) *SMSService {
	return &SMSService{
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  "http://app.mydreamstechnology.in/vb/apikey.php",
	}
}

func (s *SMSService) SendOTP(phone, otp string) error {
	message := fmt.Sprintf("Use OTP %s to log in to your Account. Never share your OTP with anyone.", otp)

	// Build URL with parameters
	params := url.Values{}
	params.Add("apikey", s.apiKey)
	params.Add("senderid", s.senderID)
	params.Add("number", phone)
	params.Add("message", message)

	fullURL := s.baseURL + "?" + params.Encode()

	// Make HTTP GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read SMS response: %v", err)
	}

	// Check if the response indicates success
	responseText := strings.ToLower(string(body))
	if !strings.Contains(responseText, "success") && resp.StatusCode != 200 {
		return fmt.Errorf("SMS sending failed: %s", string(body))
	}

	return nil
}

func (s *SMSService) SendCustomMessage(phone, message string) error {
	// Build URL with parameters
	params := url.Values{}
	params.Add("apikey", s.apiKey)
	params.Add("senderid", s.senderID)
	params.Add("number", phone)
	params.Add("message", message)

	fullURL := s.baseURL + "?" + params.Encode()

	// Make HTTP GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read SMS response: %v", err)
	}

	// Check if the response indicates success
	responseText := strings.ToLower(string(body))
	if !strings.Contains(responseText, "success") && resp.StatusCode != 200 {
		return fmt.Errorf("SMS sending failed: %s", string(body))
	}

	return nil
}
