package main

import (
	"golang-food-backend/configs"
	"golang-food-backend/internal/handlers"
	"golang-food-backend/internal/middleware"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/repositories"
	"golang-food-backend/internal/services"
	"golang-food-backend/pkg/auth"
	"golang-food-backend/pkg/cache"
	"golang-food-backend/pkg/database"
	"golang-food-backend/pkg/messaging"
	"golang-food-backend/pkg/sms"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config := configs.LoadConfig()

	// Set Gin mode
	gin.SetMode(config.Server.Mode)

	// Initialize database connections
	db, err := database.NewDatabase(config.Database.PostgresURL, config.Database.MongoURL, config.Database.MongoDBName)
	if err != nil {
		log.Fatal("Failed to connect to databases:", err)
	}
	defer db.Close()

	// Auto-migrate PostgreSQL tables
	if err := autoMigratePostgres(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize Redis cache
	redisCache := cache.NewRedisCache(config.Redis.URL, config.Redis.Password, config.Redis.DB)
	if redisCache == nil {
		log.Fatal("Failed to connect to Redis")
	}
	defer redisCache.Close()

	// Initialize Kafka
	kafkaProducer := messaging.NewKafkaProducer(config.Kafka.Brokers)
	defer kafkaProducer.Close()

	// Initialize JWT manager (access: 1 hour, refresh: 30 days)
	jwtManager := auth.NewJWTManager(config.JWT.SecretKey, config.JWT.ExpiryHours, 30)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.Postgres)
	restaurantRepo := repositories.NewRestaurantRepository(db.Postgres)
	orderRepo := repositories.NewOrderRepository(db.Postgres)
	paymentRepo := repositories.NewPaymentRepository(db.Postgres)
	cartRepo := repositories.NewCartRepository(db.Postgres)
	deliveryPartnerRepo := repositories.NewDeliveryPartnerRepository(db.Postgres)
	restaurantDeliveryPartnerRepo := repositories.NewRestaurantDeliveryPartnerRepository(db.Postgres)
	porterDeliveryRepo := repositories.NewPorterDeliveryRepository(db.Postgres)
	otpRepo := repositories.NewOTPRepository(db.Postgres) // OTP repository for SMS authentication
	// TODO: Uncomment when services are ready
	refundRepo := repositories.NewRefundRepository(db.Postgres)
	couponRepo := repositories.NewCouponRepository(db.Postgres)
	addressRepo := repositories.NewAddressRepository(db.Postgres)

	// MongoDB repositories
	productRepo := repositories.NewProductRepository(db.MongoDB)
	categoryRepo := repositories.NewProductCategoryRepository(db.MongoDB)
	// reviewRepo := repositories.NewRatingReviewRepository(db.MongoDB) // TODO: Add review service
	inventoryRepo := repositories.NewInventoryRepository(db.MongoDB)
	timeRangeProductRepo := repositories.NewTimeRangeProductRepository(db.MongoDB)

	// Initialize services
	authService := services.NewAuthService(userRepo, jwtManager, redisCache)

	// SMS and OTP services
	smsService := sms.NewSMSService("R9Jfx2ile8a6VTHu", "MYDTEH") // API credentials provided
	otpService := services.NewOTPService(otpRepo, userRepo, jwtManager, redisCache, smsService)

	restaurantService := services.NewRestaurantService(restaurantRepo)
	productService := services.NewProductService(productRepo, categoryRepo, inventoryRepo, redisCache, kafkaProducer, config.Kafka.Brokers)
	categoryService := services.NewCategoryService(categoryRepo, productRepo, redisCache)
	orderService := services.NewOrderService(orderRepo, cartRepo, paymentRepo, userRepo, inventoryRepo, redisCache, kafkaProducer, config.Kafka.Brokers)

	// Delivery and payment services
	deliveryPartnerService := services.NewDeliveryPartnerService(restaurantRepo, deliveryPartnerRepo, restaurantDeliveryPartnerRepo, orderRepo, porterDeliveryRepo)
	porterService := services.NewPorterService(orderRepo, porterDeliveryRepo)
	razorpayService := services.NewRazorpayService(config.Razorpay.KeyID, config.Razorpay.KeySecret, config.Razorpay.WebhookSecret, paymentRepo, orderRepo, deliveryPartnerService)
	// TODO: Uncomment when handlers are ready
	refundService := services.NewRefundService(refundRepo, orderRepo, paymentRepo)
	// TODO: Uncomment when handler is used: paymentService := services.NewPaymentService(paymentRepo, orderRepo, cartRepo)
	couponService := services.NewCouponService(couponRepo)
	addressService := services.NewAddressService(addressRepo)
	cartService := services.NewCartService(cartRepo, productRepo, orderRepo, paymentRepo, redisCache)
	shoptimeService := services.NewShopTimeService(restaurantRepo, timeRangeProductRepo, productRepo, redisCache)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, otpService)
	restaurantHandler := handlers.NewRestaurantHandler(restaurantService)
	productHandler := handlers.NewProductHandler(productService, categoryService)
	orderHandler := handlers.NewOrderHandler(orderService)

	// Additional handlers
	refundHandler := handlers.NewRefundHandler(refundService)
	couponHandler := handlers.NewCouponHandler(couponService)
	addressHandler := handlers.NewAddressHandler(addressService)
	cartHandler := handlers.NewCartHandler(cartService)
	shoptimeHandler := handlers.NewShopTimeHandler(shoptimeService)

	// Payment and delivery handlers
	// TODO: Uncomment when RegisterRoutes is implemented: paymentHandler := handlers.NewPaymentHandler(paymentService)
	razorpayHandler := handlers.NewRazorpayHandler(razorpayService)
	porterHandler := handlers.NewPorterHandler(porterService, porterDeliveryRepo, orderRepo, deliveryPartnerService)

	// Initialize Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(cors.Default())

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "golang-food-backend",
		})
	})

	// API routes
	api := router.Group("/api/v1")

	// Register routes
	authHandler.RegisterRoutes(api, authMiddleware)
	restaurantHandler.RegisterRoutes(api, authMiddleware)
	productHandler.RegisterRoutes(api, authMiddleware)
	orderHandler.RegisterRoutes(api, authMiddleware)

	// Additional routes
	addressHandler.RegisterRoutes(api, authMiddleware)
	couponHandler.RegisterRoutes(api, authMiddleware)
	cartHandler.RegisterRoutes(api, authMiddleware)
	refundHandler.RegisterRoutes(api, authMiddleware)
	shoptimeHandler.RegisterRoutes(api, authMiddleware)

	// Payment and delivery routes
	// TODO: Add paymentHandler.RegisterRoutes(api, authMiddleware) when RegisterRoutes is implemented
	razorpayHandler.RegisterRoutes(api)
	porterHandler.RegisterRoutes(api)

	log.Printf("🚀 Server starting on port %s", config.Server.Port)
	log.Fatal(router.Run(":" + config.Server.Port))
}

func autoMigratePostgres(db *database.Database) error {
	return db.Postgres.AutoMigrate(
		&models.User{},
		&models.Restaurant{},
		&models.RestaurantDeliveryPartners{},
		&models.RestaurantDeliveryLocationBoundary{},
		&models.CommissionToRestaurant{},
		&models.Cart{},
		&models.Order{},
		&models.OrderLog{},
		&models.Payment{},
		&models.Refund{},
		&models.DeliveryPartnerCompany{},
		&models.PorterDelivery{},
		&models.OTP{}, // Add OTP model for SMS authentication
	)
}
