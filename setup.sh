# Build script for local development
#!/bin/bash

echo "Setting up Food Delivery Backend..."

# Create necessary directories
mkdir -p logs
mkdir -p tmp

echo "✅ Food Delivery Backend setup complete!"

# Instructions for manual setup
cat << EOF

🚀 Next Steps:

1. Install Dependencies:
   go mod download

2. Set up your databases:
   - PostgreSQL: Create 'food_delivery' database
   - MongoDB: Will be auto-created
   - Redis: Start redis-server

3. Configure environment:
   cp .env.example .env
   # Edit .env with your database credentials

4. Run the application:
   go run cmd/main.go

5. Test the API:
   curl http://localhost:8080/health

📚 Documentation:
   - API endpoints: See README.md
   - Swagger docs: http://localhost:8080/api/v1/docs (when available)

🏗️ Architecture Overview:
   - PostgreSQL: Transactional data (orders, payments, users)
   - MongoDB: Flexible data (products, reviews, analytics) 
   - Redis: Caching layer
   - Kafka: Event streaming (optional for development)

EOF
