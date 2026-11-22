
package routes

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"live-polling-system/backend/controllers"
    authMiddleware "live-polling-system/backend/middleware"
)

type Dependencies struct {
	Polls     *mongo.Collection
	Users     *mongo.Collection
	Redis     *redis.Client
	JWTSecret string
}

func Register(router *gin.Engine, deps Dependencies) {

	// Validate JWT secret
	if len(deps.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET is missing or too short")
	}

	// API group
	api := router.Group("/api")

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Authentication controller
	auth := controllers.AuthController{
		Collection: deps.Users,
		JWTSecret:  deps.JWTSecret,
	}

	api.POST("/auth/signup", auth.Signup)
	api.POST("/auth/login", auth.Login)

	// Poll controller
	pollController := controllers.PollController{
		Collection: deps.Polls,
		Redis:      deps.Redis,
	}

	// Public poll listing
	api.GET("/polls", pollController.List)

	// View individual public poll
	api.GET("/polls/:id", func(c *gin.Context) {

		id := c.Param("id")

		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Invalid poll ID",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		var poll bson.M

		err = deps.Polls.FindOne(
			ctx,
			bson.M{
				"_id":       objectID,
				"is_public": true,
			},
		).Decode(&poll)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "Public poll not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Database error",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"poll":    poll,
		})
	})

	// Protected routes
	protected := api.Group("")
	protected.Use(authMiddleware.Auth(deps.JWTSecret))

	protected.POST("/polls", pollController.Create)
	protected.POST("/polls/:id/vote", pollController.Vote)
}
