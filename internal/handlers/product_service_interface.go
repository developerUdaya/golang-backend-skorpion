package handlers

import (
	"context"
	"golang-food-backend/internal/models"
	"golang-food-backend/internal/services"
)

// ProductServiceInterface defines the contract for product service
type ProductServiceInterface interface {
	CreateProduct(ctx context.Context, restaurantID string, req *services.CreateProductRequest) (*models.Product, error)
	GetProductsByRestaurant(ctx context.Context, restaurantID string, limit, offset int) ([]models.Product, error)
	SearchProducts(ctx context.Context, restaurantID, query string, limit, offset int) ([]models.Product, error)
	GetProductByID(ctx context.Context, productID string) (*models.Product, error)
	UpdateProduct(ctx context.Context, productID, restaurantID string, updates map[string]interface{}) error
	DeleteProduct(ctx context.Context, restaurantID, productID string) error
}

// CategoryServiceInterface defines the contract for category service
type CategoryServiceInterface interface {
	CreateCategory(ctx context.Context, restaurantID string, req *services.CreateCategoryRequest) (*models.ProductCategory, error)
	GetCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]models.ProductCategory, error)
	UpdateCategory(ctx context.Context, categoryID, restaurantID string, updates map[string]interface{}) error
	DeleteCategory(ctx context.Context, categoryID, restaurantID string) error
	DeleteCategoryWithProducts(ctx context.Context, categoryID, restaurantID string) error
	MoveProductsAndDeleteCategory(ctx context.Context, categoryID, targetCategoryID, restaurantID string) error
}
