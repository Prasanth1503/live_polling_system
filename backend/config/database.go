
package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var RedisClient *redis.Client

// Connect MongoDB
func ConnectMongoDB() error {

	uri := os.Getenv("MONGO_URI")

	if uri == "" {
		return fmt.Errorf("MONGO_URI is missing")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(uri),
	)

	if err != nil {
		return fmt.Errorf("MongoDB connection failed: %w", err)
	}

	// Verify MongoDB connection
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return fmt.Errorf("MongoDB ping failed: %w", err)
	}

	MongoClient = client

	fmt.Println("MongoDB connected successfully")

	return nil
}

// Connect Redis
func ConnectRedis() error {

	addr := os.Getenv("REDIS_ADDR")

	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	// Verify Redis connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("Redis connection failed: %w", err)
	}

	RedisClient = client

	fmt.Println("Redis connected successfully")

	return nil
}

// Close database connections
func DisconnectDatabases() {

	if MongoClient != nil {
		if err := MongoClient.Disconnect(context.Background()); err != nil {
			fmt.Println("MongoDB disconnect error:", err)
		}
	}

	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			fmt.Println("Redis disconnect error:", err)
		}
	}

	fmt.Println("Database connections closed")
}
