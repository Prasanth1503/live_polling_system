
package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"golang.org/x/crypto/bcrypt"

	"live-polling-system/backend/models"
)

type AuthController struct {
	Collection *mongo.Collection
	JWTSecret  string
}

// JWT Claims
type AuthClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

// Generate database context
func (ac AuthController) databaseContext(c *gin.Context) (
	context.Context,
	context.CancelFunc,
) {
	return context.WithTimeout(c.Request.Context(), 5*time.Second)
}

// Validate controller configuration
func (ac AuthController) isConfigured(c *gin.Context) bool {

	if ac.Collection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database collection is not configured",
		})
		return false
	}

	if len(ac.JWTSecret) < 32 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "JWT secret must contain at least 32 characters",
		})
		return false
	}

	return true
}

// ===============================
// SIGNUP
// ===============================

func (ac AuthController) Signup(c *gin.Context) {

	var req models.AuthRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. Enter email and password.",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	if email == "" || !strings.Contains(email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Enter a valid email",
		})
		return
	}

	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password must contain at least 8 characters",
		})
		return
	}

	// bcrypt supports passwords up to 72 bytes
	if len([]byte(req.Password)) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password must not exceed 72 bytes",
		})
		return
	}

	if !ac.isConfigured(c) {
		return
	}

	ctx, cancel := ac.databaseContext(c)
	defer cancel()

	// Check existing email
	var existing models.User

	err := ac.Collection.FindOne(ctx, bson.M{
		"email": email,
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Email already registered",
		})
		return
	}

	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Password processing failed",
		})
		return
	}

	// Create user
	user := models.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().UTC(),
	}

	result, err := ac.Collection.InsertOne(ctx, user)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already registered",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create account",
		})
		return
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)

	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	user.ID = insertedID

	// Generate JWT
	token, err := ac.createToken(user.ID.Hex())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create login token",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Signup successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}

// ===============================
// LOGIN
// ===============================

func (ac AuthController) Login(c *gin.Context) {

	var req models.AuthRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	if email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email and password are required",
		})
		return
	}

	if !ac.isConfigured(c) {
		return
	}

	ctx, cancel := ac.databaseContext(c)
	defer cancel()

	// Find user
	var user models.User

	err := ac.Collection.FindOne(ctx, bson.M{
		"email": email,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error",
		})
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Generate token
	token, err := ac.createToken(user.ID.Hex())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create login token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"email": user.Email,
		},
	})
}

// ===============================
// GENERATE JWT TOKEN
// ===============================

func (ac AuthController) createToken(userID string) (string, error) {

	claims := AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString([]byte(ac.JWTSecret))
}
