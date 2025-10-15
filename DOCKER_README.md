# Docker Setup Instructions

## Prerequisites

- Docker and Docker Compose installed
- Git (to clone the repository)

## Quick Start

1. **Clone the repository** (if not already done):
   ```bash
   git clone https://github.com/developerUdaya/golang-backend-skorpion.git
   cd golang-backend-skorpion
   ```

2. **Create environment file**:
   ```bash
   cp .env.example .env
   ```
   
   Update the `.env` file with your actual API keys and secrets.

3. **Build and run with Docker Compose**:
   ```bash
   docker-compose up --build
   ```

4. **Wait for all services to be healthy**:
   The application will start once all dependencies (PostgreSQL, MongoDB, Redis, Kafka) are ready.

5. **Access the application**:
   - API: http://localhost:8080
   - Health check: http://localhost:8080/health

## Services Included

- **PostgreSQL** (port 5432): Primary database for user management, orders, etc.
- **MongoDB** (port 27017): Document database for products, reviews, etc.  
- **Redis** (port 6379): Caching and session storage
- **Kafka** (port 9092): Message queue for events
- **Zookeeper** (port 2181): Kafka coordination
- **Go Application** (port 8080): Main backend service

## Development Commands

### Run in development mode:
```bash
docker-compose up --build
```

### Run in background:
```bash
docker-compose up -d --build
```

### View logs:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f app
docker-compose logs -f mongodb
docker-compose logs -f postgres
```

### Stop services:
```bash
docker-compose down
```

### Stop and remove volumes (⚠️ This will delete all data):
```bash
docker-compose down -v
```

### Rebuild application only:
```bash
docker-compose build app
docker-compose up app
```

## Database Access

### PostgreSQL:
```bash
# Connect to PostgreSQL container
docker exec -it golang-food-postgres psql -U postgres -d golang_food_db

# Or use external tool with:
# Host: localhost
# Port: 5432
# Username: postgres
# Password: postgres123
# Database: golang_food_db
```

### MongoDB:
```bash
# Connect to MongoDB container
docker exec -it golang-food-mongodb mongosh -u admin -p admin123

# Or use MongoDB Compass with:
# URI: mongodb://admin:admin123@localhost:27017
```

### Redis:
```bash
# Connect to Redis container
docker exec -it golang-food-redis redis-cli -a redis123
```

## API Testing

Once the application is running, you can test the APIs:

### Health Check:
```bash
curl http://localhost:8080/health
```

### Register a customer:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "email": "test@example.com",
    "phone": "+1234567890", 
    "password": "password123",
    "role": "customer",
    "restaurant_id": "123e4567-e89b-12d3-a456-426614174000"
  }'
```

## Troubleshooting

### Application won't start:
1. Check if all dependencies are healthy:
   ```bash
   docker-compose ps
   ```

2. View application logs:
   ```bash
   docker-compose logs app
   ```

### Database connection issues:
1. Ensure database containers are running:
   ```bash
   docker-compose ps postgres mongodb
   ```

2. Check database logs:
   ```bash
   docker-compose logs postgres
   docker-compose logs mongodb
   ```

### Port conflicts:
If you have services running on the default ports, you can modify the ports in `docker-compose.yml`:
```yaml
services:
  postgres:
    ports:
      - "5433:5432"  # Change left side to available port
```

### Clean restart:
```bash
docker-compose down -v
docker system prune -f
docker-compose up --build
```

## Production Considerations

Before deploying to production:

1. **Update secrets** in the environment variables
2. **Use proper SSL certificates**
3. **Configure firewall rules**
4. **Set up monitoring and logging**
5. **Use external managed databases** for better reliability
6. **Enable backup strategies**

## Environment Variables

Key environment variables to configure:

- `JWT_SECRET`: Strong secret for JWT tokens
- `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`: Razorpay payment credentials  
- `PORTER_API_KEY`: Porter delivery API key
- `SMS_API_KEY`: SMS service API key
- Database passwords and connection strings

See `.env.example` for complete list.