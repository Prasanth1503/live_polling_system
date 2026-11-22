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

func ConnectMongoDB() error {

    uri := os.Getenv("MONGO_URI")

    if uri == "" {
        return fmt.Errorf("MONGO_URI is missing")
    }

    ctx, cancel := context.WithTimeout(
        context.Background(), 10*time.Second,
    )
    defer cancel()

    client, err := mongo.Connect(
        ctx,
        options.Client().ApplyURI(uri),
    )

    if err != nil {
        return err
    }

    if err := client.Ping(ctx, nil); err != nil {
        return err
    }

    MongoClient = client

    fmt.Println("MongoDB connected successfully")

    return nil
}

func ConnectRedis() error {

    addr := os.Getenv("REDIS_ADDR")

    if addr == "" {
        addr = "localhost:6379"
    }

    RedisClient = redis.NewClient(&redis.Options{
        Addr: addr,
    })

    ctx, cancel := context.WithTimeout(
        context.Background(), 5*time.Second,
    )
    defer cancel()

    if err := RedisClient.Ping(ctx).Err(); err != nil {
        return err
    }

    fmt.Println("Redis connected successfully")

    return nil
}
