package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Poll struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Question   string             `bson:"question" json:"question"`
	Options    []string           `bson:"options" json:"options"`
	Counts     []int64            `bson:"counts" json:"counts"`
	TotalVotes int64              `bson:"totalVotes" json:"totalVotes"`
	OwnerID    string 			  `bson:"ownerId" json:"ownerId"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}

type CreatePollRequest struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

type VoteRequest struct {
	OptionIndex int `json:"optionIndex"`
}