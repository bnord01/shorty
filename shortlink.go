package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Shortlink object as stored in the MongoDB
type Shortlink struct {
	ID          primitive.ObjectID `json:"-"`
	ShortUrl    string             `json:"short" bson:"short"`
	LongUrl     string             `json:"long" bson:"long"`
	Description string             `json:"descr" bson:"descr"`
	AccessCount int                `json:"access_count" bson:"access_count"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Shortlink Update struct
type ShortlinkUpdate struct {
	ShortUrl    string    `json:"short" bson:"short"`
	LongUrl     string    `json:"long" bson:"long"`
	Description string    `json:"descr" bson:"descr"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}
