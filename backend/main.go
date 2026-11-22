
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	userCollection *mongo.Collection
	pollCollection *mongo.Collection
	jwtSecret      []byte
)

// ================= USER MODEL =================

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	PasswordHash string             `json:"-" bson:"passwordHash"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

// ================= POLL MODEL =================

type Poll struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Question   string             `json:"question" bson:"question"`
	Options    []string           `json:"options" bson:"options"`
	Counts     []int              `json:"counts" bson:"counts"`
	TotalVotes int                `json:"totalVotes" bson:"totalVotes"`
	OwnerID    string             `json:"-" bson:"ownerId"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

// ================= RESPONSE HELPERS =================

func successResponse(c *gin.Context, status int, data gin.H) {
	data["success"] = true
	c.JSON(status, data)
}

func errorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   message,
	})
}

// ================= SIGNUP =================

func Signup(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request")
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || len(input.Password) < 8 {
		errorResponse(c, http.StatusBadRequest,
			"Email is required and password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Password processing failed")
		return
	}

	user := User{
		Email:        input.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	result, err := userCollection.InsertOne(c.Request.Context(), user)
	if err != nil {
		errorResponse(c, http.StatusConflict, "Email may already be registered")
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)

	token, err := createToken(user.ID.Hex())
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not create token")
		return
	}

	successResponse(c, http.StatusCreated, gin.H{
		"message": "Signup successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}

// ================= LOGIN =================

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid request")
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var user User

	err := userCollection.FindOne(
		c.Request.Context(),
		bson.M{"email": input.Email},
	).Decode(&user)

	if err != nil {
		errorResponse(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	)

	if err != nil {
		errorResponse(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	token, err := createToken(user.ID.Hex())
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not create token")
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}

// ================= JWT =================

func createToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtSecret)
}

// ================= AUTH MIDDLEWARE =================

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		header := c.GetHeader("Authorization")

		if !strings.HasPrefix(header, "Bearer ") {
			errorResponse(c, http.StatusUnauthorized, "Authorization token required")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return jwtSecret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)

		if err != nil || token == nil || !token.Valid {
			errorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			errorResponse(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
			return
		}

		userID, ok := claims["userId"].(string)
		if !ok || userID == "" {
			errorResponse(c, http.StatusUnauthorized, "Invalid user")
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

// ================= CREATE POLL =================

func CreatePoll(c *gin.Context) {

	var input struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid poll data")
		return
	}

	input.Question = strings.TrimSpace(input.Question)

	if input.Question == "" {
		errorResponse(c, http.StatusBadRequest, "Question is required")
		return
	}

	if len(input.Options) < 2 || len(input.Options) > 6 {
		errorResponse(c, http.StatusBadRequest, "Provide between 2 and 6 options")
		return
	}

	for i := range input.Options {
		input.Options[i] = strings.TrimSpace(input.Options[i])

		if input.Options[i] == "" {
			errorResponse(c, http.StatusBadRequest, "Options cannot be empty")
			return
		}
	}

	userID, exists := c.Get("userID")
	if !exists {
		errorResponse(c, http.StatusUnauthorized, "Login required")
		return
	}

	poll := Poll{
		Question:   input.Question,
		Options:    input.Options,
		Counts:     make([]int, len(input.Options)),
		TotalVotes: 0,
		OwnerID:    userID.(string),
		CreatedAt:  time.Now(),
	}

	result, err := pollCollection.InsertOne(
		c.Request.Context(),
		poll,
	)

	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not create poll")
		return
	}

	poll.ID = result.InsertedID.(primitive.ObjectID)

	successResponse(c, http.StatusCreated, gin.H{
		"message": "Poll created successfully",
		"poll":    poll,
		"shareUrl": "http://localhost:5173/poll/" + poll.ID.Hex(),
	})
}

// ================= ADMIN POLLS =================

func GetPolls(c *gin.Context) {

	userID, exists := c.Get("userID")
	if !exists {
		errorResponse(c, http.StatusUnauthorized, "Login required")
		return
	}

	cursor, err := pollCollection.Find(
		c.Request.Context(),
		bson.M{"ownerId": userID.(string)},
		options.Find().SetSort(
			bson.D{{Key: "createdAt", Value: -1}},
		),
	)

	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not load polls")
		return
	}

	defer cursor.Close(c.Request.Context())

	polls := []Poll{}

	if err := cursor.All(c.Request.Context(), &polls); err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not read polls")
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"polls": polls,
	})
}

// ================= PUBLIC POLL =================

func GetPublicPoll(c *gin.Context) {

	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid poll ID")
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	var poll Poll

	err = pollCollection.FindOne(
		ctx,
		bson.M{"_id": objectID},
	).Decode(&poll)

	if err != nil {

		if err == mongo.ErrNoDocuments {
			errorResponse(c, http.StatusNotFound, "Poll not found")
			return
		}

		errorResponse(c, http.StatusInternalServerError, "Could not load poll")
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"poll": poll,
	})
}

// ================= PUBLIC VOTING =================

func SubmitPublicVote(c *gin.Context) {

	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid poll ID")
		return
	}

	var input struct {
		OptionIndex int `json:"optionIndex"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid vote data")
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	var poll Poll

	err = pollCollection.FindOne(
		ctx,
		bson.M{"_id": objectID},
	).Decode(&poll)

	if err != nil {

		if err == mongo.ErrNoDocuments {
			errorResponse(c, http.StatusNotFound, "Poll not found")
			return
		}

		errorResponse(c, http.StatusInternalServerError, "Could not load poll")
		return
	}

	if input.OptionIndex < 0 ||
		input.OptionIndex >= len(poll.Options) {
		errorResponse(c, http.StatusBadRequest, "Invalid option selected")
		return
	}

	if len(poll.Counts) != len(poll.Options) {
		errorResponse(c, http.StatusInternalServerError, "Poll vote data is inconsistent")
		return
	}

	updatePath := fmt.Sprintf("counts.%d", input.OptionIndex)

	result, err := pollCollection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
			updatePath: bson.M{"$exists": true},
		},
		bson.M{
			"$inc": bson.M{
				updatePath:  1,
				"totalVotes": 1,
			},
		},
	)

	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not save vote")
		return
	}

	if result.MatchedCount == 0 {
		errorResponse(c, http.StatusConflict, "Poll vote data changed; reload the poll")
		return
	}

	// Load updated poll results
	err = pollCollection.FindOne(
		ctx,
		bson.M{"_id": objectID},
	).Decode(&poll)

	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "Could not load updated results")
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"message": "Vote submitted successfully",
		"poll":    poll,
	})
}

// ================= HEALTH CHECK =================

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "ok",
		"message": "Backend is running",
	})
}

// ================= MAIN =================

func main() {

	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "live_polling"
	}

	secret := os.Getenv("JWT_SECRET")

	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must contain at least 32 characters")
	}

	jwtSecret = []byte(secret)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error: ", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping error: ", err)
	}

	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer disconnectCancel()

		_ = mongoClient.Disconnect(disconnectCtx)
	}()

	log.Println("MongoDB connected successfully")

	db := mongoClient.Database(dbName)

	userCollection = db.Collection("users")
	pollCollection = db.Collection("polls")

	// Unique email index
	_, err = userCollection.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	)

	if err != nil {
		log.Fatal("Could not create email index: ", err)
	}

	// Gin router
	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Authorization",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API routes
	api := router.Group("/api")

	api.GET("/health", Health)

	// Authentication
	api.POST("/auth/signup", Signup)
	api.POST("/auth/login", Login)

	// Public routes - no login
	api.GET("/polls/:id", GetPublicPoll)
	api.POST("/polls/:id/vote", SubmitPublicVote)

	// Protected routes - login required
	protected := api.Group("")
	protected.Use(AuthMiddleware())

	protected.POST("/polls", CreatePoll)
	protected.GET("/polls", GetPolls)

	// Print all registered routes
	for _, route := range router.Routes() {
		log.Printf("REGISTERED: %s %s", route.Method, route.Path)
	}

	log.Println("Go server running on port " + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Server error: ", err)
	}
}