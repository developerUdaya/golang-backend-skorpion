package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	Postgres *gorm.DB
	MongoDB  *mongo.Database
}

func NewDatabase(postgresURL, mongoURL, mongoDBName string) (*Database, error) {
	// Initialize PostgreSQL
	postgresDB, err := initPostgreSQL(postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Initialize MongoDB (but make it optional)
	var mongoDB *mongo.Database
	mongoDB, err = initMongoDB(mongoURL, mongoDBName)
	if err != nil {
		log.Printf("Warning: MongoDB connection failed: %v. Some features may not work.", err)
		// Continue without MongoDB
		mongoDB = nil
	}

	return &Database{
		Postgres: postgresDB,
		MongoDB:  mongoDB,
	}, nil
}

func initPostgreSQL(url string) (*gorm.DB, error) {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(url), config)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connected to PostgreSQL successfully")
	return db, nil
}

func initMongoDB(url, dbName string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		return nil, err
	}

	// Test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB successfully")
	return client.Database(dbName), nil
}

func (db *Database) Close() error {
	// Close PostgreSQL
	if sqlDB, err := db.Postgres.DB(); err == nil {
		sqlDB.Close()
	}

	// Close MongoDB
	if db.MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return db.MongoDB.Client().Disconnect(ctx)
	}

	return nil
}
