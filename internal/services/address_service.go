package services

import (
	"context"
	"errors"

	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"

	"github.com/google/uuid"
)

type AddressService struct {
	addressRepo repositories.AddressRepository
}

func NewAddressService(addressRepo repositories.AddressRepository) *AddressService {
	return &AddressService{
		addressRepo: addressRepo,
	}
}

// Request and Response types
type CreateAddressRequest struct {
	Type        string  `json:"type" binding:"required,oneof=home work other"`
	AddressLine string  `json:"address_line" binding:"required"`
	Landmark    string  `json:"landmark"`
	City        string  `json:"city" binding:"required"`
	State       string  `json:"state" binding:"required"`
	PostalCode  string  `json:"postal_code" binding:"required"`
	Country     string  `json:"country" binding:"required"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	IsDefault   bool    `json:"is_default"`
}

type UpdateAddressRequest struct {
	Type        string  `json:"type" binding:"omitempty,oneof=home work other"`
	AddressLine string  `json:"address_line"`
	Landmark    string  `json:"landmark"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	PostalCode  string  `json:"postal_code"`
	Country     string  `json:"country"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	IsDefault   *bool   `json:"is_default"`
}

type AddressListResponse struct {
	Addresses  []models.Address `json:"addresses"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	TotalPages int              `json:"total_pages"`
}

func (s *AddressService) CreateAddress(ctx context.Context, userID string, req *CreateAddressRequest) (*models.Address, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// If this is set as default, unset other default addresses
	if req.IsDefault {
		if err := s.addressRepo.UnsetDefaultAddresses(ctx, userUUID); err != nil {
			return nil, err
		}
	}

	// Create address
	address := &models.Address{
		UserID:       userUUID,
		Type:         req.Type,
		AddressLine1: req.AddressLine,
		AddressLine2: req.Landmark,
		City:         req.City,
		State:        req.State,
		Country:      req.Country,
		PinCode:      req.PostalCode,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		IsDefault:    req.IsDefault,
	}

	if err := s.addressRepo.Create(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}

func (s *AddressService) GetAddresses(ctx context.Context, userID string, page, limit int) (*AddressListResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	offset := (page - 1) * limit

	addresses, total, err := s.addressRepo.GetByUserID(ctx, userUUID, offset, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &AddressListResponse{
		Addresses:  addresses,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

func (s *AddressService) GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	id, err := uuid.Parse(addressID)
	if err != nil {
		return nil, errors.New("invalid address ID")
	}

	address, err := s.addressRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify address belongs to user
	if address.UserID != userUUID {
		return nil, errors.New("address does not belong to user")
	}

	return address, nil
}

func (s *AddressService) UpdateAddress(ctx context.Context, userID, addressID string, req *UpdateAddressRequest) (*models.Address, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	id, err := uuid.Parse(addressID)
	if err != nil {
		return nil, errors.New("invalid address ID")
	}

	address, err := s.addressRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify address belongs to user
	if address.UserID != userUUID {
		return nil, errors.New("address does not belong to user")
	}

	// Update fields
	if req.Type != "" {
		address.Type = req.Type
	}
	if req.AddressLine != "" {
		address.AddressLine1 = req.AddressLine
	}
	if req.Landmark != "" {
		address.AddressLine2 = req.Landmark
	}
	if req.City != "" {
		address.City = req.City
	}
	if req.State != "" {
		address.State = req.State
	}
	if req.PostalCode != "" {
		address.PinCode = req.PostalCode
	}
	if req.Country != "" {
		address.Country = req.Country
	}
	if req.Latitude != 0 {
		address.Latitude = req.Latitude
	}
	if req.Longitude != 0 {
		address.Longitude = req.Longitude
	}
	if req.IsDefault != nil {
		// If setting as default, unset other default addresses
		if *req.IsDefault {
			if err := s.addressRepo.UnsetDefaultAddresses(ctx, userUUID); err != nil {
				return nil, err
			}
		}
		address.IsDefault = *req.IsDefault
	}

	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}

func (s *AddressService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	id, err := uuid.Parse(addressID)
	if err != nil {
		return errors.New("invalid address ID")
	}

	address, err := s.addressRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify address belongs to user
	if address.UserID != userUUID {
		return errors.New("address does not belong to user")
	}

	return s.addressRepo.Delete(ctx, id)
}

func (s *AddressService) SetDefaultAddress(ctx context.Context, userID, addressID string) (*models.Address, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	id, err := uuid.Parse(addressID)
	if err != nil {
		return nil, errors.New("invalid address ID")
	}

	address, err := s.addressRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify address belongs to user
	if address.UserID != userUUID {
		return nil, errors.New("address does not belong to user")
	}

	// Unset other default addresses
	if err := s.addressRepo.UnsetDefaultAddresses(ctx, userUUID); err != nil {
		return nil, err
	}

	// Set this address as default
	address.IsDefault = true
	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}
