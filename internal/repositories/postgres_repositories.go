package repositories

import (
	"context"
	"golang-food-backend/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Multi-restaurant support: get user by email within specific restaurant
func (r *userRepository) GetByEmailAndRestaurant(ctx context.Context, email string, restaurantID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ? AND restaurant_id = ?", email, restaurantID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Multi-restaurant support: get user by phone within specific restaurant
func (r *userRepository) GetByPhoneAndRestaurant(ctx context.Context, phone string, restaurantID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ? AND restaurant_id = ?", phone, restaurantID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

func (r *userRepository) GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Where("restaurant_id = ?", restaurantID).Find(&users).Error
	return users, err
}

// Restaurant Repository
type restaurantRepository struct {
	db *gorm.DB
}

func NewRestaurantRepository(db *gorm.DB) RestaurantRepository {
	return &restaurantRepository{db: db}
}

func (r *restaurantRepository) Create(ctx context.Context, restaurant *models.Restaurant) error {
	return r.db.WithContext(ctx).Create(restaurant).Error
}

func (r *restaurantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Restaurant, error) {
	var restaurant models.Restaurant
	err := r.db.WithContext(ctx).Preload("Owner").Where("id = ?", id).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

func (r *restaurantRepository) Update(ctx context.Context, restaurant *models.Restaurant) error {
	return r.db.WithContext(ctx).Save(restaurant).Error
}

func (r *restaurantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Restaurant{}, id).Error
}

func (r *restaurantRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&restaurants).Error
	return restaurants, err
}

func (r *restaurantRepository) Search(ctx context.Context, query string, limit, offset int) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	err := r.db.WithContext(ctx).
		Where("name ILIKE ? OR description ILIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(limit).Offset(offset).Find(&restaurants).Error
	return restaurants, err
}

func (r *restaurantRepository) GetRestaurantsWithAutoOpenClose() ([]*models.Restaurant, error) {
	var restaurants []*models.Restaurant
	err := r.db.Where("auto_open_close = ? AND status = ?", true, "active").Find(&restaurants).Error
	return restaurants, err
}

// Order Repository
type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Restaurant").
		Preload("Cart").
		Preload("PorterDeliveries", "is_active = ?", true).
		Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) Update(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *orderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Order{}, id).Error
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Preload("Restaurant").
		Preload("PorterDeliveries", "is_active = ?", true).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

func (r *orderRepository) GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID, limit, offset int) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("PorterDeliveries", "is_active = ?", true).
		Where("restaurant_id = ?", restaurantID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

func (r *orderRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Restaurant").
		Where("order_status = ?", status).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

// Payment Repository
type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *paymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.WithContext(ctx).Where("transaction_id = ?", transactionID).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

func (r *paymentRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.WithContext(ctx).
		Preload("Order").
		Preload("User").
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&payments).Error
	return payments, err
}

// Cart Repository
type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) Create(ctx context.Context, cart *models.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

func (r *cartRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Cart, error) {
	var cart models.Cart
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Restaurant").
		Where("id = ?", id).First(&cart).Error
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Cart, error) {
	var cart models.Cart
	err := r.db.WithContext(ctx).
		Preload("Restaurant").
		Where("user_id = ? AND status = ?", userID, "active").First(&cart).Error
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) Update(ctx context.Context, cart *models.Cart) error {
	return r.db.WithContext(ctx).Save(cart).Error
}

func (r *cartRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Cart{}, id).Error
}

// Coupon repository implementation
type couponRepository struct {
	db *gorm.DB
}

func NewCouponRepository(db *gorm.DB) CouponRepository {
	return &couponRepository{db: db}
}

func (r *couponRepository) Create(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Create(coupon).Error
}

func (r *couponRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&coupon).Error
	if err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (r *couponRepository) GetByCode(ctx context.Context, code string) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&coupon).Error
	if err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (r *couponRepository) Update(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Save(coupon).Error
}

func (r *couponRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Coupon{}, id).Error
}

func (r *couponRepository) GetCouponsWithFilters(ctx context.Context, offset, limit int, restaurantID *uuid.UUID, active *bool) ([]models.Coupon, int64, error) {
	var coupons []models.Coupon
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Coupon{})

	// Apply filters
	if restaurantID != nil {
		query = query.Where("restaurant_id = ?", *restaurantID)
	}
	if active != nil {
		query = query.Where("is_active = ?", *active)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := query.Offset(offset).Limit(limit).Find(&coupons).Error; err != nil {
		return nil, 0, err
	}

	return coupons, total, nil
}

// Refund repository implementation
type refundRepository struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) RefundRepository {
	return &refundRepository{db: db}
}

func (r *refundRepository) Create(ctx context.Context, refund *models.Refund) error {
	return r.db.WithContext(ctx).Create(refund).Error
}

func (r *refundRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Refund, error) {
	var refund models.Refund
	err := r.db.WithContext(ctx).
		Preload("Order").
		Preload("Payment").
		Preload("User").
		Where("id = ?", id).First(&refund).Error
	if err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *refundRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Refund, error) {
	var refund models.Refund
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&refund).Error
	if err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *refundRepository) Update(ctx context.Context, refund *models.Refund) error {
	return r.db.WithContext(ctx).Save(refund).Error
}

func (r *refundRepository) GetByUserIDWithFilters(ctx context.Context, userID uuid.UUID, offset, limit int, status string) ([]models.Refund, int64, error) {
	var refunds []models.Refund
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Refund{}).Where("user_id = ?", userID)

	// Apply status filter
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with preloaded relations
	if err := query.Preload("Order").Preload("Payment").
		Offset(offset).Limit(limit).Find(&refunds).Error; err != nil {
		return nil, 0, err
	}

	return refunds, total, nil
}

// Address repository implementation
type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(ctx context.Context, address *models.Address) error {
	return r.db.WithContext(ctx).Create(address).Error
}

func (r *addressRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Address, error) {
	var address models.Address
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&address).Error
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Address, int64, error) {
	var addresses []models.Address
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Address{}).Where("user_id = ?", userID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results ordered by is_default desc, created_at desc
	if err := query.Order("is_default DESC, created_at DESC").
		Offset(offset).Limit(limit).Find(&addresses).Error; err != nil {
		return nil, 0, err
	}

	return addresses, total, nil
}

func (r *addressRepository) Update(ctx context.Context, address *models.Address) error {
	return r.db.WithContext(ctx).Save(address).Error
}

func (r *addressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Address{}, id).Error
}

func (r *addressRepository) UnsetDefaultAddresses(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Address{}).
		Where("user_id = ?", userID).
		Update("is_default", false).Error
}

// Delivery Partner Repository
type deliveryPartnerRepository struct {
	db *gorm.DB
}

func NewDeliveryPartnerRepository(db *gorm.DB) DeliveryPartnerRepository {
	return &deliveryPartnerRepository{db: db}
}

func (r *deliveryPartnerRepository) Create(ctx context.Context, partner *models.DeliveryPartnerCompany) error {
	return r.db.WithContext(ctx).Create(partner).Error
}

func (r *deliveryPartnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryPartnerCompany, error) {
	var partner models.DeliveryPartnerCompany
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&partner).Error
	if err != nil {
		return nil, err
	}
	return &partner, nil
}

func (r *deliveryPartnerRepository) Update(ctx context.Context, partner *models.DeliveryPartnerCompany) error {
	return r.db.WithContext(ctx).Save(partner).Error
}

func (r *deliveryPartnerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.DeliveryPartnerCompany{}, id).Error
}

func (r *deliveryPartnerRepository) GetAll(ctx context.Context, limit, offset int) ([]models.DeliveryPartnerCompany, error) {
	var partners []models.DeliveryPartnerCompany
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&partners).Error
	return partners, err
}

func (r *deliveryPartnerRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.DeliveryPartnerCompany, error) {
	var partners []models.DeliveryPartnerCompany
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&partners).Error
	return partners, err
}

// Restaurant Delivery Partner Repository
type restaurantDeliveryPartnerRepository struct {
	db *gorm.DB
}

func NewRestaurantDeliveryPartnerRepository(db *gorm.DB) RestaurantDeliveryPartnerRepository {
	return &restaurantDeliveryPartnerRepository{db: db}
}

func (r *restaurantDeliveryPartnerRepository) Create(ctx context.Context, relationship *models.RestaurantDeliveryPartners) error {
	return r.db.WithContext(ctx).Create(relationship).Error
}

func (r *restaurantDeliveryPartnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.RestaurantDeliveryPartners, error) {
	var relationship models.RestaurantDeliveryPartners
	err := r.db.WithContext(ctx).
		Preload("Restaurant").
		Preload("DeliveryPartnerCompany").
		Where("id = ?", id).First(&relationship).Error
	if err != nil {
		return nil, err
	}
	return &relationship, nil
}

func (r *restaurantDeliveryPartnerRepository) GetByRestaurantID(ctx context.Context, restaurantID uuid.UUID) ([]models.RestaurantDeliveryPartners, error) {
	var relationships []models.RestaurantDeliveryPartners
	err := r.db.WithContext(ctx).
		Preload("DeliveryPartnerCompany").
		Where("restaurant_id = ?", restaurantID).Find(&relationships).Error
	return relationships, err
}

func (r *restaurantDeliveryPartnerRepository) GetByDeliveryPartnerID(ctx context.Context, partnerID uuid.UUID) ([]models.RestaurantDeliveryPartners, error) {
	var relationships []models.RestaurantDeliveryPartners
	err := r.db.WithContext(ctx).
		Preload("Restaurant").
		Where("delivery_partner_company_id = ?", partnerID).Find(&relationships).Error
	return relationships, err
}

func (r *restaurantDeliveryPartnerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.RestaurantDeliveryPartners{}, id).Error
}

// PorterDeliveryRepository implementation
type porterDeliveryRepository struct {
	db *gorm.DB
}

func NewPorterDeliveryRepository(db *gorm.DB) PorterDeliveryRepository {
	return &porterDeliveryRepository{db: db}
}

func (r *porterDeliveryRepository) Create(ctx context.Context, delivery *models.PorterDelivery) error {
	return r.db.WithContext(ctx).Create(delivery).Error
}

func (r *porterDeliveryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PorterDelivery, error) {
	var delivery models.PorterDelivery
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&delivery).Error
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *porterDeliveryRepository) GetByPorterOrderID(ctx context.Context, porterOrderID string) (*models.PorterDelivery, error) {
	var delivery models.PorterDelivery
	err := r.db.WithContext(ctx).Where("porter_order_id = ?", porterOrderID).First(&delivery).Error
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *porterDeliveryRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.PorterDelivery, error) {
	var deliveries []models.PorterDelivery
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&deliveries).Error
	return deliveries, err
}

func (r *porterDeliveryRepository) GetActiveByOrderID(ctx context.Context, orderID uuid.UUID) (*models.PorterDelivery, error) {
	var delivery models.PorterDelivery
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND is_active = ?", orderID, true).
		First(&delivery).Error
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *porterDeliveryRepository) Update(ctx context.Context, delivery *models.PorterDelivery) error {
	return r.db.WithContext(ctx).Save(delivery).Error
}

func (r *porterDeliveryRepository) UpdateStatus(ctx context.Context, porterOrderID string, status string, metadata map[string]interface{}) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Add specific fields based on status
	if metadata != nil {
		if partnerName, ok := metadata["partner_name"]; ok {
			updates["partner_name"] = partnerName
		}
		if partnerPhone, ok := metadata["partner_phone"]; ok {
			updates["partner_phone_number"] = partnerPhone
		}
		if partnerPicture, ok := metadata["partner_picture"]; ok {
			updates["partner_picture"] = partnerPicture
		}
		if vehicleType, ok := metadata["vehicle_type"]; ok {
			updates["vehicle_type"] = vehicleType
		}
		if vehicleNumber, ok := metadata["vehicle_number"]; ok {
			updates["vehicle_number"] = vehicleNumber
		}
		if trackingURL, ok := metadata["tracking_url"]; ok {
			updates["tracking_url"] = trackingURL
		}
		if estimatedTime, ok := metadata["estimated_delivery_time"]; ok {
			updates["estimated_delivery_time"] = estimatedTime
		}
		if actualTime, ok := metadata["actual_delivery_time"]; ok {
			updates["actual_delivery_time"] = actualTime
		}
		if pickupTime, ok := metadata["pickup_time"]; ok {
			updates["pickup_time"] = pickupTime
		}

		// Update status history
		updates["status_history"] = gorm.Expr("COALESCE(status_history, '[]'::jsonb) || ?",
			models.JSONB{
				"status":    status,
				"timestamp": "NOW()",
				"metadata":  metadata,
			})
	}

	return r.db.WithContext(ctx).
		Model(&models.PorterDelivery{}).
		Where("porter_order_id = ?", porterOrderID).
		Updates(updates).Error
}

func (r *porterDeliveryRepository) DeactivateOldDeliveries(ctx context.Context, orderID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.PorterDelivery{}).
		Where("order_id = ? AND is_active = ?", orderID, true).
		Update("is_active", false).Error
}

// OTP Repository Implementation
type otpRepository struct {
	db *gorm.DB
}

func NewOTPRepository(db *gorm.DB) OTPRepository {
	return &otpRepository{db: db}
}

func (r *otpRepository) Create(ctx context.Context, otp *models.OTP) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *otpRepository) GetValidOTP(ctx context.Context, phone string, restaurantID uuid.UUID, otpCode string) (*models.OTP, error) {
	var otp models.OTP
	err := r.db.WithContext(ctx).Where(
		"phone = ? AND restaurant_id = ? AND otp_code = ? AND expires_at > ? AND is_used = false AND attempt_count < ?",
		phone, restaurantID, otpCode, time.Now(), 5, // Max 5 attempts
	).First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *otpRepository) GetValidOTPWithOptionalRestaurant(ctx context.Context, phone string, restaurantID *uuid.UUID, otpCode string) (*models.OTP, error) {
	var otp models.OTP
	query := r.db.WithContext(ctx).Where(
		"phone = ? AND otp_code = ? AND expires_at > ? AND is_used = false AND attempt_count < ?",
		phone, otpCode, time.Now(), 5, // Max 5 attempts
	)

	if restaurantID != nil {
		query = query.Where("restaurant_id = ?", *restaurantID)
	} else {
		query = query.Where("restaurant_id IS NULL")
	}

	err := query.First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *otpRepository) InvalidateOTP(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.OTP{}).Where("id = ?", id).Update("is_used", true).Error
}

func (r *otpRepository) DeleteExpiredOTPs(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.OTP{}).Error
}

func (r *otpRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.OTP{}).Where("id = ?", id).UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error
}
