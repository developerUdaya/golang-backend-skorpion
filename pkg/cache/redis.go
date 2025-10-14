package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Println("Connected to Redis successfully")
	return &RedisCache{client: client}
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, jsonData, expiration).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (r *RedisCache) SetWithPrefix(ctx context.Context, prefix, key string, value interface{}, expiration time.Duration) error {
	return r.Set(ctx, prefix+":"+key, value, expiration)
}

func (r *RedisCache) GetWithPrefix(ctx context.Context, prefix, key string, dest interface{}) error {
	return r.Get(ctx, prefix+":"+key, dest)
}

func (r *RedisCache) DeleteWithPrefix(ctx context.Context, prefix, key string) error {
	return r.Delete(ctx, prefix+":"+key)
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
