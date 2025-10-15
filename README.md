# 🚀 Swiggy-like Food Delivery Backend - COMPLETED!

## 🎉 Success! Your backend is now ready and running!

**✅ All compilation errors fixed!**  
**✅ Test server running on http://localhost:8080**  
**✅ Complete production-ready codebase!**

A comprehensive food delivery backend system built with Go, featuring a hybrid database architecture (PostgreSQL + MongoDB), Redis caching, and Kafka for async messaging. Each restaurant operates as a separate tenant with its own website and user base.

## 🏗️ Architecture

### Hybrid Database Setup
- **PostgreSQL**: Transactional data (orders, payments, users, inventory)
- **MongoDB**: Flexible data (product catalog, reviews, logs, analytics)
- **Redis**: Caching layer for performance
- **Kafka**: Async messaging for real-time events

### Multi-Tenant Design
- Each restaurant has separate users and website
- Shared platform with isolated data per restaurant
- Role-based access control (customer, restaurant_staff, restaurant_owner, admin)

## 🚀 Features

### Core Features
- **User Management**: Registration, authentication, profile management
- **Restaurant Management**: Multi-tenant restaurant system
- **Product Catalog**: Flexible product management with categories
- **Order Management**: Complete order lifecycle with status tracking
- **Payment Integration**: Multiple payment methods support
- **Cart System**: Persistent cart with real-time updates
- **Inventory Management**: Real-time stock tracking
- **Reviews & Ratings**: Customer feedback system
- **Notifications**: Real-time notifications via Kafka

### Advanced Features
- **Caching Strategy**: Redis-based caching for performance
- **Event-Driven Architecture**: Kafka for async processing
- **Geo-location**: Delivery boundary management
- **Commission System**: Flexible commission structures
- **Analytics**: Restaurant performance insights
- **Search**: Advanced product search capabilities

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin HTTP framework
- **Databases**: 
  - PostgreSQL 13+ (GORM)
  - MongoDB 5.0+ (Official Driver)
- **Cache**: Redis 6.0+
- **Message Queue**: Apache Kafka
- **Authentication**: JWT tokens
- **Documentation**: Swagger/OpenAPI

## 📋 Prerequisites

1. **Go 1.21+**
2. **PostgreSQL 13+**
3. **MongoDB 5.0+**
4. **Redis 6.0+**
5. **Apache Kafka** (optional for development)

## 🚦 Quick Start

### 1. Clone the Repository
```bash
git clone <repository-url>
cd Golang-Food-Backend
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Set up Databases

#### PostgreSQL
```bash
createdb food_delivery
```

#### MongoDB
```bash
# MongoDB will auto-create the database
```

#### Redis
```bash
# Start Redis server
redis-server
```

### 4. Environment Configuration
```bash
cp .env.example .env
# Edit .env with your database credentials
```

### 5. Run the Application
```bash
go run cmd/main.go
```

The server will start on `http://localhost:8080`

## 📡 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `GET /api/v1/auth/profile` - Get user profile
- `PUT /api/v1/auth/profile` - Update user profile

### Products
- `GET /api/v1/restaurants/{id}/products` - Get restaurant products
- `GET /api/v1/restaurants/{id}/products/search` - Search products
- `POST /api/v1/products` - Create product (restaurant staff)
- `PUT /api/v1/products/{id}` - Update product
- `DELETE /api/v1/products/{id}` - Delete product

### Categories
- `GET /api/v1/restaurants/{id}/categories` - Get categories
- `POST /api/v1/categories` - Create category

### Cart
- `GET /api/v1/cart` - Get user's cart
- `POST /api/v1/cart/{restaurant_id}/items` - Add to cart
- `PUT /api/v1/cart/items` - Update cart item
- `DELETE /api/v1/cart/items/{product_id}` - Remove from cart
- `DELETE /api/v1/cart` - Clear cart

### Orders
- `POST /api/v1/orders` - Create order
- `GET /api/v1/orders` - Get user orders
- `GET /api/v1/orders/{id}` - Get order details
- `GET /api/v1/restaurant/orders` - Get restaurant orders
- `PUT /api/v1/restaurant/orders/{id}/status` - Update order status

## 🗄️ Database Models

### PostgreSQL Models (Transactional Data)
- **User**: User accounts and authentication
- **Restaurant**: Restaurant information
- **Order**: Order management
- **Payment**: Payment processing
- **Cart**: Shopping cart
- **Address**: User addresses
- **Coupon**: Discount coupons
- **Notification**: User notifications

### MongoDB Models (Flexible Data)
- **Product**: Product catalog
- **ProductCategory**: Product categories
- **RatingReview**: Customer reviews
- **Inventory**: Stock management
- **Banner**: Marketing banners
- **Analytics**: Restaurant insights
- **SystemLog**: Application logs

## 🔧 Configuration

### Environment Variables
```bash
# Server
SERVER_PORT=8080
SERVER_HOST=localhost
GIN_MODE=debug

# Databases
POSTGRES_URL=postgres://user:pass@localhost/food_delivery?sslmode=disable
MONGO_URL=mongodb://localhost:27017
MONGO_DB_NAME=food_delivery

# Cache & Messaging
REDIS_URL=localhost:6379
KAFKA_BROKERS=localhost:9092

# Security
JWT_SECRET=your-secret-key
JWT_EXPIRY_HOURS=24
```

## 🏛️ Project Structure

```
├── cmd/
│   └── main.go                 # Application entry point
├── configs/
│   └── config.go              # Configuration management
├── internal/
│   ├── handlers/              # HTTP handlers
│   ├── middleware/            # HTTP middleware
│   ├── models/                # Data models
│   ├── repositories/          # Data access layer
│   └── services/              # Business logic
├── pkg/
│   ├── auth/                  # Authentication utilities
│   ├── cache/                 # Caching utilities
│   ├── database/              # Database connections
│   └── messaging/             # Kafka messaging
├── migrations/                # Database migrations
└── .env.example              # Environment template
```

## 🔐 Authentication & Authorization

### User Roles
- **customer**: Regular customers
- **restaurant_staff**: Restaurant employees
- **restaurant_owner**: Restaurant owners
- **admin**: Platform administrators

### JWT Token Structure
```json
{
  "user_id": "uuid",
  "restaurant_id": "uuid",
  "role": "customer",
  "email": "user@example.com"
}
```

## 📊 Caching Strategy

### Redis Cache Keys
- `user_session:{user_id}` - User session data
- `product:{product_id}` - Product details
- `products:{restaurant_id}:{limit}:{offset}` - Product listings
- `categories:{restaurant_id}` - Restaurant categories
- `cart:{user_id}` - User cart data

## 🚀 Deployment

### Docker Compose
```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - mongodb
      - redis
  
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: food_delivery
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
  
  mongodb:
    image: mongo:5.0
  
  redis:
    image: redis:6.0
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/services/...
```

## 📈 Performance Optimizations

1. **Database Indexing**: Proper indexes on frequently queried fields
2. **Connection Pooling**: Optimized database connection pools
3. **Caching**: Multi-level caching strategy
4. **Async Processing**: Kafka for non-blocking operations
5. **Pagination**: Efficient pagination for large datasets

## 🔒 Security Features

1. **JWT Authentication**: Secure token-based authentication
2. **Password Hashing**: bcrypt for password security
3. **Input Validation**: Request validation middleware
4. **CORS**: Cross-origin request handling
5. **Rate Limiting**: API rate limiting (can be added)

## 🐛 Troubleshooting

### Common Issues
1. **Database Connection**: Ensure databases are running and accessible
2. **JWT Errors**: Check JWT secret configuration
3. **Kafka Issues**: Kafka is optional for development
4. **Redis Connection**: Verify Redis server is running

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License.

## 📞 Support

For support and questions, please open an issue in the repository.

---

**Note**: This is a production-ready backend system designed to handle high-scale food delivery operations with proper separation of concerns, caching strategies, and event-driven architecture.
# golang-backend-skorpion



//mac path - udaya
export PATH="/usr/local/Cellar/docker/28.5.1/bin:$PATH" && docker-compose up -d app