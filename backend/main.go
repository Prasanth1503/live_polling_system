package main

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	userCollection *mongo.Collection
	pollCollection *mongo.Collection
	jwtSecret      []byte
	frontendURL    string
)

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	PasswordHash string             `json:"-" bson:"passwordHash"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

type Poll struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Question   string             `json:"question" bson:"question"`
	Options    []string           `json:"options" bson:"options"`
	Counts     []int              `json:"counts" bson:"counts"`
	TotalVotes int                `json:"totalVotes" bson:"totalVotes"`
	OwnerID    string             `json:"-" bson:"ownerId"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

func success(c *gin.Context, status int, data gin.H) {
	data["success"] = true
	c.JSON(status, data)
}

func failure(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   message,
	})
}

func createToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func Signup(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		failure(c, 400, "Invalid request data")
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if !strings.Contains(input.Email, "@") || len(input.Email) < 5 {
		failure(c, 400, "Enter a valid email")
		return
	}

	if len(input.Password) < 8 {
		failure(c, 400, "Password must contain at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		failure(c, 500, "Password processing failed")
		return
	}

	user := User{
		Email:        input.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			failure(c, 409, "Email already registered")
			return
		}
		log.Println("Signup error:", err)
		failure(c, 500, "Could not register user")
		return
	}

	id := result.InsertedID.(primitive.ObjectID)

	token, err := createToken(id.Hex())
	if err != nil {
		failure(c, 500, "Could not create token")
		return
	}

	success(c, 201, gin.H{
		"message": "Signup successful",
		"token":   token,
		"user": gin.H{
			"id":    id.Hex(),
			"email": user.Email,
		},
	})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		failure(c, 400, "Invalid request data")
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user User

	err := userCollection.FindOne(
		ctx,
		bson.M{"email": input.Email},
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			failure(c, 401, "Invalid email or password")
			return
		}
		log.Println("Login error:", err)
		failure(c, 500, "Database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	); err != nil {
		failure(c, 401, "Invalid email or password")
		return
	}

	token, err := createToken(user.ID.Hex())
	if err != nil {
		failure(c, 500, "Could not create token")
		return
	}

	success(c, 200, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		if !strings.HasPrefix(header, "Bearer ") {
			failure(c, 401, "Authorization token required")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))

		token, err := jwt.Parse(
			tokenString,
			func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return jwtSecret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)

		if err != nil || token == nil || !token.Valid {
			failure(c, 401, "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			failure(c, 401, "Invalid token claims")
			c.Abort()
			return
		}

		userID, ok := claims["userId"].(string)
		if !ok {
			failure(c, 401, "Invalid user")
			c.Abort()
			return
		}

		if _, err := primitive.ObjectIDFromHex(userID); err != nil {
			failure(c, 401, "Invalid user ID")
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func CreatePoll(c *gin.Context) {
	var input struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		failure(c, 400, "Invalid poll data")
		return
	}

	input.Question = strings.TrimSpace(input.Question)

	if input.Question == "" {
		failure(c, 400, "Question is required")
		return
	}

	if len(input.Options) < 2 || len(input.Options) > 6 {
		failure(c, 400, "Provide between 2 and 6 options")
		return
	}

	for i := range input.Options {
		input.Options[i] = strings.TrimSpace(input.Options[i])
		if input.Options[i] == "" {
			failure(c, 400, "Options cannot be empty")
			return
		}
	}

	userID, exists := c.Get("userID")
	if !exists {
		failure(c, 401, "Login required")
		return
	}

	poll := Poll{
		Question:  input.Question,
		Options:   input.Options,
		Counts:    make([]int, len(input.Options)),
		OwnerID:   userID.(string),
		CreatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := pollCollection.InsertOne(ctx, poll)
	if err != nil {
		log.Println("Create poll error:", err)
		failure(c, 500, "Could not create poll")
		return
	}

	poll.ID = result.InsertedID.(primitive.ObjectID)

	success(c, 201, gin.H{
		"message": "Poll created successfully",
		"poll":    poll,
		"shareUrl": frontendURL +
			"/poll/" + poll.ID.Hex(),
	})
}

func GetPolls(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		failure(c, 401, "Login required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := pollCollection.Find(
		ctx,
		bson.M{"ownerId": userID.(string)},
		options.Find().SetSort(
			bson.D{{Key: "createdAt", Value: -1}},
		),
	)

	if err != nil {
		log.Println("Get polls error:", err)
		failure(c, 500, "Could not load polls")
		return
	}
	defer cursor.Close(ctx)

	polls := []Poll{}
	if err := cursor.All(ctx, &polls); err != nil {
		failure(c, 500, "Could not read polls")
		return
	}

	success(c, 200, gin.H{"polls": polls})
}

func GetPublicPoll(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		failure(c, 400, "Invalid poll ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var poll Poll
	err = pollCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&poll)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			failure(c, 404, "Poll not found")
			return
		}
		log.Println("Get public poll error:", err)
		failure(c, 500, "Could not load poll")
		return
	}

	success(c, 200, gin.H{"poll": poll})
}

func SubmitPublicVote(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		failure(c, 400, "Invalid poll ID")
		return
	}

	var input struct {
		OptionIndex int `json:"optionIndex"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		failure(c, 400, "Invalid vote data")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var poll Poll
	err = pollCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&poll)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			failure(c, 404, "Poll not found")
			return
		}
		failure(c, 500, "Could not load poll")
		return
	}

	if input.OptionIndex < 0 || input.OptionIndex >= len(poll.Options) {
		failure(c, 400, "Invalid option selected")
		return
	}

	if len(poll.Counts) != len(poll.Options) {
		failure(c, 500, "Poll vote data is inconsistent")
		return
	}

	path := fmt.Sprintf("counts.%d", input.OptionIndex)

	result, err := pollCollection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{
				path:         1,
				"totalVotes": 1,
			},
		},
	)

	if err != nil {
		log.Println("Vote update error:", err)
		failure(c, 500, "Could not save vote")
		return
	}

	if result.MatchedCount == 0 {
		failure(c, 404, "Poll not found")
		return
	}

	if err := pollCollection.FindOne(
		ctx,
		bson.M{"_id": id},
	).Decode(&poll); err != nil {
		failure(c, 500, "Could not load updated results")
		return
	}

	success(c, 200, gin.H{
		"message": "Vote submitted successfully",
		"poll":    poll,
	})
}

func Health(c *gin.Context) {
	success(c, 200, gin.H{
		"status":  "ok",
		"message": "Backend is running",
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	switch ginMode {
	case gin.DebugMode, gin.ReleaseMode, gin.TestMode:
		gin.SetMode(ginMode)
	default:
		log.Fatal("GIN_MODE must be debug, release, or test")
	}

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

	frontendURL = os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5174"
	}
	frontendURL = strings.TrimRight(frontendURL, "/")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		cancel()
		log.Fatal("MongoDB connection error:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		cancel()
		log.Fatal("MongoDB ping error:", err)
	}
	cancel()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Println("MongoDB disconnect error:", err)
		}
	}()

	log.Println("MongoDB connected successfully")

	db := client.Database(dbName)
	userCollection = db.Collection("users")
	pollCollection = db.Collection("polls")

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = userCollection.Indexes().CreateOne(
		indexCtx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)
	indexCancel()

	if err != nil {
		log.Fatal("Email index creation error:", err)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatal("Trusted proxy configuration error:", err)
	}

	router.Use(
		gin.Logger(),
		gin.Recovery(),
		cors.New(cors.Config{
			AllowOrigins: []string{
				frontendURL,
				"http://localhost:5174",
				"http://127.0.0.1:5174",
			},
			AllowMethods: []string{
				"GET",
				"POST",
				"PUT",
				"DELETE",
				"OPTIONS",
			},
			AllowHeaders: []string{
				"Origin",
				"Content-Type",
				"Authorization",
			},
			AllowCredentials: true,
		}),
	)

	api := router.Group("/api")
	api.GET("/health", Health)
	api.POST("/auth/signup", Signup)
	api.POST("/auth/login", Login)

	api.GET("/polls/:id", GetPublicPoll)
	api.POST("/polls/:id/vote", SubmitPublicVote)

	protected := api.Group("/")
	protected.Use(AuthMiddleware())
	protected.POST("/polls", CreatePoll)
	protected.GET("/polls", GetPolls)

	log.Println("Backend URL: http://localhost:" + port)
	log.Println("Health URL: http://localhost:" + port + "/api/health")

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Server error:", err)
	}
}
