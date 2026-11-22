package controllers

import (
	"context"
	"net/http"
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

func (pc PollController) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	cursor, err := pc.Collection.Find(ctx, bson.M{})
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"Could not fetch polls"}); return }
	defer cursor.Close(ctx)
	polls := make([]models.Poll, 0)
	if err := cursor.All(ctx, &polls); err != nil { c.JSON(500, gin.H{"error":"Could not decode polls"}); return }
	// Redis stores the live counters. Read them when building the response.
	for i := range polls {
		for j := range polls[i].Counts {
			value, err := pc.Redis.Get(ctx, countKey(polls[i].ID.Hex(), j)).Int64()
			if err == nil { polls[i].Counts[j] = value }
		}
		polls[i].TotalVotes = 0
		for _, count := range polls[i].Counts { polls[i].TotalVotes += count }
	}
	c.JSON(http.StatusOK, polls)
}

func (pc PollController) Create(c *gin.Context) {
	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":"Invalid JSON"}); return }
	req.Question = strings.TrimSpace(req.Question)
	if len(req.Question) < 5 || len(req.Question) > 200 { c.JSON(400, gin.H{"error":"Question must be 5–200 characters"}); return }
	if len(req.Options) < 2 || len(req.Options) > 6 { c.JSON(400, gin.H{"error":"Provide 2–6 options"}); return }
	for i := range req.Options {
		req.Options[i] = strings.TrimSpace(req.Options[i])
		if req.Options[i] == "" || len(req.Options[i]) > 80 { c.JSON(400, gin.H{"error":"Each option must be 1–80 characters"}); return }
	}
	poll := models.Poll{
	Question:  req.Question,
	Options:   req.Options,
	Counts:    make([]int64, len(req.Options)),
	OwnerID:   c.GetString("userID"),
	CreatedAt: time.Now().UTC(),
    }
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second); defer cancel()
	result, err := pc.Collection.InsertOne(ctx, poll)
	if err != nil { c.JSON(500, gin.H{"error":"Could not save poll"}); return }
	poll.ID = result.InsertedID.(primitive.ObjectID)
	pipe := pc.Redis.Pipeline()
	for i := range poll.Options { pipe.Set(ctx, countKey(poll.ID.Hex(), i), 0, 0) }
	if _, err := pipe.Exec(ctx); err != nil { c.JSON(500, gin.H{"error":"Poll saved but counter initialization failed"}); return }
	c.JSON(http.StatusCreated, poll)
}

func (pc PollController) Vote(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil { c.JSON(400, gin.H{"error":"Invalid poll ID"}); return }
	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":"Invalid vote"}); return }
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second); defer cancel()
	var poll models.Poll
	if err := pc.Collection.FindOne(ctx, bson.M{"_id":id}).Decode(&poll); err != nil { c.JSON(404, gin.H{"error":"Poll not found"}); return }
	if req.OptionIndex < 0 || req.OptionIndex >= len(poll.Options) { c.JSON(400, gin.H{"error":"Option index is out of range"}); return }
	key := countKey(id.Hex(), req.OptionIndex)
	count, err := pc.Redis.Incr(ctx, key).Result()
	if err != nil { c.JSON(500, gin.H{"error":"Could not record vote"}); return }
	// Persist a snapshot in MongoDB as a durable record of the current totals.
	if _, err := pc.Collection.UpdateOne(ctx, bson.M{"_id":id}, bson.M{"$set":bson.M{"counts."+itoa(req.OptionIndex):count}, "$inc":bson.M{"totalVotes":1}}); err != nil {
		c.JSON(500, gin.H{"error":"Vote counted in Redis but MongoDB update failed"}); return
	}
	_ = pc.Redis.Publish(ctx, "poll:"+id.Hex(), countKeyPayload(req.OptionIndex, count)).Err()
	c.JSON(http.StatusOK, gin.H{"message":"Vote recorded","count":count})
}

func countKey(pollID string, option int) string { return "poll:"+pollID+":option:"+itoa(option) }
func itoa(n int) string {
	if n == 0 { return "0" }
	var b [20]byte; i:=len(b)
	for n>0 { i--; b[i]=byte('0'+n%10); n/=10 }
	return string(b[i:])
}
func countKeyPayload(option int, count int64) string { return `{"optionIndex":`+itoa(option)+`,"count":`+itoa64(count)+`}` }
func itoa64(n int64) string { if n==0{return "0"}; var b [20]byte; i:=len(b); for n>0{i--;b[i]=byte('0'+n%10);n/=10}; return string(b[i:]) }
