package repositories

import (
	"context"
	"golang-food-backend/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Product Repository
type productRepository struct {
	collection *mongo.Collection
}

func NewProductRepository(db *mongo.Database) ProductRepository {
	return &productRepository{
		collection: db.Collection("products"),
	}
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}
	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Product, error) {
	var product models.Product
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *models.Product) error {
	product.UpdatedAt = time.Now()

	filter := bson.M{"_id": product.ID}
	update := bson.M{"$set": product}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *productRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

func (r *productRepository) GetByRestaurantID(ctx context.Context, restaurantID string, limit, offset int) ([]models.Product, error) {
	var products []models.Product

	filter := bson.M{"restaurant_id": restaurantID, "is_available": true}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) GetByCategoryID(ctx context.Context, categoryID primitive.ObjectID, limit, offset int) ([]models.Product, error) {
	var products []models.Product

	filter := bson.M{"category_id": categoryID, "is_available": true}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) Search(ctx context.Context, query string, restaurantID string, limit, offset int) ([]models.Product, error) {
	var products []models.Product

	filter := bson.M{
		"restaurant_id": restaurantID,
		"is_available":  true,
		"$or": []bson.M{
			{"name": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$in": []string{query}}},
		},
	}

	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) GetHighlighted(ctx context.Context, restaurantID string, highlightType string) ([]models.Product, error) {
	// This would typically involve aggregation with the HighlightProduct collection
	// For now, we'll return products based on tags
	var products []models.Product

	filter := bson.M{
		"restaurant_id": restaurantID,
		"is_available":  true,
		"tags":          bson.M{"$in": []string{highlightType}},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

// ProductCategory Repository
type productCategoryRepository struct {
	collection *mongo.Collection
}

func NewProductCategoryRepository(db *mongo.Database) ProductCategoryRepository {
	return &productCategoryRepository{
		collection: db.Collection("product_categories"),
	}
}

func (r *productCategoryRepository) Create(ctx context.Context, category *models.ProductCategory) error {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, category)
	if err != nil {
		return err
	}
	category.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *productCategoryRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.ProductCategory, error) {
	var category models.ProductCategory
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&category)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *productCategoryRepository) Update(ctx context.Context, category *models.ProductCategory) error {
	category.UpdatedAt = time.Now()

	filter := bson.M{"_id": category.ID}
	update := bson.M{"$set": category}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *productCategoryRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

func (r *productCategoryRepository) GetByRestaurantID(ctx context.Context, restaurantID string) ([]models.ProductCategory, error) {
	var categories []models.ProductCategory

	filter := bson.M{"restaurant_id": restaurantID, "is_active": true}
	opts := options.Find().SetSort(bson.D{{"sort_order", 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

// RatingReview Repository
type ratingReviewRepository struct {
	collection *mongo.Collection
}

func NewRatingReviewRepository(db *mongo.Database) RatingReviewRepository {
	return &ratingReviewRepository{
		collection: db.Collection("rating_reviews"),
	}
}

func (r *ratingReviewRepository) Create(ctx context.Context, review *models.RatingReview) error {
	review.CreatedAt = time.Now()
	review.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, review)
	if err != nil {
		return err
	}
	review.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ratingReviewRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.RatingReview, error) {
	var review models.RatingReview
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&review)
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *ratingReviewRepository) Update(ctx context.Context, review *models.RatingReview) error {
	review.UpdatedAt = time.Now()

	filter := bson.M{"_id": review.ID}
	update := bson.M{"$set": review}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *ratingReviewRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

func (r *ratingReviewRepository) GetByEntityID(ctx context.Context, entityID string, reviewType string, limit, offset int) ([]models.RatingReview, error) {
	var reviews []models.RatingReview

	filter := bson.M{
		"entity_id":   entityID,
		"review_type": reviewType,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (r *ratingReviewRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]models.RatingReview, error) {
	var reviews []models.RatingReview

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

// Inventory Repository
type inventoryRepository struct {
	collection *mongo.Collection
}

func NewInventoryRepository(db *mongo.Database) InventoryRepository {
	return &inventoryRepository{
		collection: db.Collection("inventory"),
	}
}

func (r *inventoryRepository) Create(ctx context.Context, inventory *models.Inventory) error {
	inventory.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, inventory)
	if err != nil {
		return err
	}
	inventory.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *inventoryRepository) GetByProductID(ctx context.Context, productID primitive.ObjectID) (*models.Inventory, error) {
	var inventory models.Inventory
	err := r.collection.FindOne(ctx, bson.M{"product_id": productID}).Decode(&inventory)
	if err != nil {
		return nil, err
	}
	return &inventory, nil
}

func (r *inventoryRepository) Update(ctx context.Context, inventory *models.Inventory) error {
	inventory.UpdatedAt = time.Now()

	filter := bson.M{"_id": inventory.ID}
	update := bson.M{"$set": inventory}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *inventoryRepository) UpdateQuantity(ctx context.Context, productID primitive.ObjectID, quantity int) error {
	filter := bson.M{"product_id": productID}
	update := bson.M{
		"$set": bson.M{
			"quantity":   quantity,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *inventoryRepository) ReserveStock(ctx context.Context, productID primitive.ObjectID, quantity int) error {
	filter := bson.M{"product_id": productID}
	update := bson.M{
		"$inc": bson.M{"reserved_quantity": quantity},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *inventoryRepository) ReleaseStock(ctx context.Context, productID primitive.ObjectID, quantity int) error {
	filter := bson.M{"product_id": productID}
	update := bson.M{
		"$inc": bson.M{"reserved_quantity": -quantity},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *inventoryRepository) GetLowStock(ctx context.Context, restaurantID string) ([]models.Inventory, error) {
	var inventories []models.Inventory

	// Find products where current quantity is below minimum stock level
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"restaurant_id": restaurantID,
				"$expr": bson.M{
					"$lte": []string{"$quantity", "$min_stock_level"},
				},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &inventories); err != nil {
		return nil, err
	}

	return inventories, nil
}

// TimeRangeProduct Repository
type timeRangeProductRepository struct {
	groupCollection *mongo.Collection
	itemCollection  *mongo.Collection
}

func NewTimeRangeProductRepository(db *mongo.Database) TimeRangeProductRepository {
	return &timeRangeProductRepository{
		groupCollection: db.Collection("time_range_products_groups"),
		itemCollection:  db.Collection("time_range_products_group_items"),
	}
}

func (r *timeRangeProductRepository) CreateTimeGroup(ctx context.Context, group *models.TimeRangeProductsGroup) error {
	group.ID = primitive.NewObjectID()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	_, err := r.groupCollection.InsertOne(ctx, group)
	return err
}

func (r *timeRangeProductRepository) GetTimeGroupByID(ctx context.Context, id primitive.ObjectID) (*models.TimeRangeProductsGroup, error) {
	var group models.TimeRangeProductsGroup
	filter := bson.M{"_id": id}

	err := r.groupCollection.FindOne(ctx, filter).Decode(&group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (r *timeRangeProductRepository) GetTimeGroupsByRestaurant(ctx context.Context, restaurantID string) ([]models.TimeRangeProductsGroup, error) {
	var groups []models.TimeRangeProductsGroup

	filter := bson.M{
		"restaurant_id": restaurantID,
		"is_active":     true,
	}

	opts := options.Find().SetSort(bson.D{{"start_time", 1}})

	cursor, err := r.groupCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (r *timeRangeProductRepository) UpdateTimeGroup(ctx context.Context, group *models.TimeRangeProductsGroup) error {
	group.UpdatedAt = time.Now()

	filter := bson.M{"_id": group.ID}
	update := bson.M{"$set": group}

	_, err := r.groupCollection.UpdateOne(ctx, filter, update)
	return err
}

func (r *timeRangeProductRepository) DeleteTimeGroup(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.groupCollection.DeleteOne(ctx, filter)
	return err
}

func (r *timeRangeProductRepository) AddProductToTimeGroup(ctx context.Context, item *models.TimeRangeProductsGroupItem) error {
	item.ID = primitive.NewObjectID()
	item.CreatedAt = time.Now()

	_, err := r.itemCollection.InsertOne(ctx, item)
	return err
}

func (r *timeRangeProductRepository) RemoveProductFromTimeGroup(ctx context.Context, groupID, productID primitive.ObjectID) error {
	filter := bson.M{
		"group_id":   groupID,
		"product_id": productID,
	}

	_, err := r.itemCollection.DeleteOne(ctx, filter)
	return err
}

func (r *timeRangeProductRepository) GetProductsByTimeGroup(ctx context.Context, groupID primitive.ObjectID) ([]models.TimeRangeProductsGroupItem, error) {
	var items []models.TimeRangeProductsGroupItem

	filter := bson.M{"group_id": groupID}

	cursor, err := r.itemCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *timeRangeProductRepository) GetActiveProductsByTime(ctx context.Context, restaurantID string, currentTime string) ([]primitive.ObjectID, error) {
	var productIDs []primitive.ObjectID

	// Find active time groups for current time
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"restaurant_id": restaurantID,
				"is_active":     true,
				"start_time":    bson.M{"$lte": currentTime},
				"end_time":      bson.M{"$gte": currentTime},
			}},
		},
		{
			{"$lookup", bson.M{
				"from":         "time_range_products_group_items",
				"localField":   "_id",
				"foreignField": "group_id",
				"as":           "products",
			}},
		},
		{
			{"$unwind", "$products"},
		},
		{
			{"$project", bson.M{
				"product_id": "$products.product_id",
			}},
		},
	}

	cursor, err := r.groupCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result struct {
			ProductID primitive.ObjectID `bson:"product_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		productIDs = append(productIDs, result.ProductID)
	}

	return productIDs, nil
}
