
package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"live-polling-system/backend/models"
)

type PollController struct {
	Collection *mongo.Collection
	Redis      *redis.Client
}

// Check database connections
func (pc PollController) configured(c *gin.Context) bool {

	if pc.Collection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Poll database is not configured",
		})
		return false
	}

	if pc.Redis == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Redis is not configured",
		})
		return false
	}

	return true
}

// ===============================
// LIST POLLS
// ===============================

func (pc PollController) List(c *gin.Context) {

	if !pc.configured(c) {
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	cursor, err := pc.Collection.Find(ctx, bson.M{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not fetch polls",
		})
		return
	}

	defer cursor.Close(ctx)

	polls := make([]models.Poll, 0)

	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not decode polls",
		})
		return
	}

	// Read live counters from Redis
	for i := range polls {

		if len(polls[i].Counts) != len(polls[i].Options) {
			polls[i].Counts = make([]int64, len(polls[i].Options))
		}

		polls[i].TotalVotes = 0

		for j := range polls[i].Options {

			key := countKey(polls[i].ID.Hex(), j)

			value, err := pc.Redis.Get(ctx, key).Int64()

			if err == nil {
				polls[i].Counts[j] = value
			}

			// Redis unavailable or key missing:
			// use MongoDB stored count.
			polls[i].TotalVotes += polls[i].Counts[j]
		}
	}

	c.JSON(http.StatusOK, polls)
}

// ===============================
// CREATE POLL
// ===============================

func (pc PollController) Create(c *gin.Context) {

	if !pc.configured(c) {
		return
	}

	var req models.CreatePollRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid poll request",
		})
		return
	}

	req.Question = strings.TrimSpace(req.Question)

	if len(req.Question) < 5 || len(req.Question) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Question must be 5-200 characters",
		})
		return
	}

	if len(req.Options) < 2 || len(req.Options) > 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide 2-6 options",
		})
		return
	}

	// Validate options
	for i := range req.Options {

		req.Options[i] = strings.TrimSpace(req.Options[i])

		if req.Options[i] == "" || len(req.Options[i]) > 80 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Each option must be 1-80 characters",
			})
			return
		}
	}

	// Get authenticated user
	ownerID := c.GetString("userID")

	if ownerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Login required to create a poll",
		})
		return
	}

	poll := models.Poll{
		Question:  req.Question,
		Options:   req.Options,
		Counts:    make([]int64, len(req.Options)),
		OwnerID:   ownerID,
		CreatedAt: time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	// Save poll in MongoDB
	result, err := pc.Collection.InsertOne(ctx, poll)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not save poll",
		})
		return
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)

	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid poll ID",
		})
		return
	}

	poll.ID = insertedID

	// Initialize Redis counters
	pipe := pc.Redis.Pipeline()

	for i := range poll.Options {

		pipe.Set(
			ctx,
			countKey(poll.ID.Hex(), i),
			0,
			0,
		)
	}

	if _, err := pipe.Exec(ctx); err != nil {

		c.JSON(http.StatusCreated, gin.H{
			"message": "Poll created, but live counters need initialization",
			"poll":    poll,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Poll created successfully",
		"poll":    poll,
	})
}

// ===============================
// VOTE
// ===============================

func (pc PollController) Vote(c *gin.Context) {

	if !pc.configured(c) {
		return
	}

	id, err := primitive.ObjectIDFromHex(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid poll ID",
		})
		return
	}

	var req models.VoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid vote request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	// Find poll
	var poll models.Poll

	err = pc.Collection.FindOne(
		ctx,
		bson.M{"_id": id},
	).Decode(&poll)

	if err != nil {

		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Poll not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not fetch poll",
		})
		return
	}

	// Validate option index
	if req.OptionIndex < 0 ||
		req.OptionIndex >= len(poll.Options) {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Option index is out of range",
		})
		return
	}

	// Ensure counts array is initialized
	if len(poll.Counts) != len(poll.Options) {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Poll counters are not initialized correctly",
		})
		return
	}

	key := countKey(id.Hex(), req.OptionIndex)

	// Increment Redis counter
	count, err := pc.Redis.Incr(ctx, key).Result()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not record vote in Redis",
		})
		return
	}

	// Increment MongoDB counter atomically
	field := "counts." + strconv.Itoa(req.OptionIndex)

	_, err = pc.Collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{
				field:      1,
				"totalVotes": 1,
			},
		},
	)

	if err != nil {

		// Best-effort rollback of Redis increment
		_, _ = pc.Redis.Decr(ctx, key).Result()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not save vote in database",
		})
		return
	}

	// Publish live update
	payload, _ := json.Marshal(gin.H{
		"optionIndex": req.OptionIndex,
		"count":       count,
		"totalVotes":  poll.TotalVotes + 1,
	})

	_ = pc.Redis.Publish(
		ctx,
		"poll:"+id.Hex(),
		string(payload),
	).Err()

	c.JSON(http.StatusOK, gin.H{
		"message":     "Vote recorded successfully",
		"optionIndex": req.OptionIndex,
		"count":       count,
	})
}

// ===============================
// REDIS KEY
// ===============================

func countKey(pollID string, option int) string {

	return "poll:" + pollID +
		":option:" + strconv.Itoa(option)
}
