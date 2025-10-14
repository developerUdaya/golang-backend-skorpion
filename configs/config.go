package configs

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
	Razorpay RazorpayConfig
	Porter   PorterConfig
}

type ServerConfig struct {
	Port string
	Host string
	Mode string
}

type DatabaseConfig struct {
	PostgresURL string
	MongoURL    string
	MongoDBName string
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

type JWTConfig struct {
	SecretKey   string
	ExpiryHours int
}

type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	WebhookSecret string
}

type PorterConfig struct {
	APIKey  string
	BaseURL string
}

// 10 digit mobile
// app android/ios
// no api access token
// 1250 per agent/per month


func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "localhost"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			PostgresURL: getEnv("POSTGRES_URL", "postgres://user:password@localhost:5432/food_delivery?sslmode=disable"),
			MongoURL:    getEnv("MONGO_URL", "mongodb://localhost:27017"),
			MongoDBName: getEnv("MONGO_DB_NAME", "food_delivery"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "food-delivery-service"),
		},
		JWT: JWTConfig{
			SecretKey:   getEnv("JWT_SECRET", "your-secret-key"),
			ExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
		},
		Razorpay: RazorpayConfig{
			KeyID:         getEnv("RAZORPAY_KEY_ID", "rzp_test_key"),
			KeySecret:     getEnv("RAZORPAY_KEY_SECRET", "secret"),
			WebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", "webhook_secret"),
		},
		Porter: PorterConfig{
			APIKey:  getEnv("PORTER_API_KEY", "O8AJTXXXXXXXXXX-UA1LiA"),
			BaseURL: getEnv("PORTER_BASE_URL", "https://pfe-apigw-uat.porter.in"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
