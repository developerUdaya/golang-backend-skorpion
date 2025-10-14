#!/bin/bash

# Simple configuration for running without external dependencies
export SERVER_PORT=8080
export SERVER_HOST=localhost
export GIN_MODE=debug

export POSTGRES_URL=postgres://postgres:postgres@localhost:5432/food_delivery?sslmode=disable
export MONGO_URL=mongodb://localhost:27017
export MONGO_DB_NAME=food_delivery

export REDIS_URL=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

export KAFKA_BROKERS=localhost:9092
export KAFKA_GROUP_ID=food-delivery-service

export JWT_SECRET=your-super-secret-jwt-key-for-development
export JWT_EXPIRY_HOURS=24

export RAZORPAY_KEY_ID=rzp_test_key
export RAZORPAY_KEY_SECRET=rzp_test_secret
export RAZORPAY_WEBHOOK_SECRET=webhook_secret

echo "Starting application with modified test configuration..."
echo "Note: Using PostgreSQL only mode and disabling MongoDB temporarily"

# Compile and run with MongoDB disabled flag
go run -ldflags="-X 'golang-food-backend/internal/config.DisableMongoDB=true'" cmd/main.go