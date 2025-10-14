#!/bin/bash

# Simple test configuration for running without external dependencies
export DATABASE_HOST="localhost"
export DATABASE_USER="test"
export DATABASE_PASSWORD="test"
export DATABASE_NAME="test"
export DATABASE_PORT="5432"
export DATABASE_SSLMODE="disable"

export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DATABASE="test"

export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DB="0"

export JWT_SECRET_KEY="your-super-secret-jwt-key-here"
export JWT_EXPIRY_HOURS="24"

export KAFKA_BROKERS="localhost:9092"

echo "Starting application with test configuration..."
echo "Note: Database connections will fail without running services"
echo "To install Docker and run with real databases:"
echo "brew install docker"
echo "docker compose -f docker-compose.dev.yml up -d"
echo ""

# Try to run the application
go run cmd/main.go
