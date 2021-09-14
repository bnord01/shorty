package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// Timeout for database operations
var timeout = 5

// Default mongo database to use
var default_db = "shorty"

// Default collection in the mongo database
var default_collection = "shorts"

// Shared mongo collection of shortlinks
// Safe to be used by multiple goroutines according to https://github.com/mongodb/mongo-go-driver/blob/33fac989d3a3f042cd94b5aa3400accc0fac04a3/mongo/collection.go#L30
var coll *mongo.Collection

// Connects to the MongoDB and sets up the shared collection `coll`
// This sets up a shutdown hook on SIGTERM/SIGINT to disconnect the database if the program is interrupted.
// The caller must make sure to disconnect the client manually via `coll.Database().Client().Disconnect(ctx)`
// if the program is to terminate otherwise.
func Connect() error {

	// Get connection data from the environment
	connectionURI := os.Getenv("MONGO_URL")

	db_name := os.Getenv("SHORTY_DB")
	if db_name == "" {
		db_name = default_db
	}

	coll_name := os.Getenv("SHORTY_COLLECTION")
	if coll_name == "" {
		coll_name = default_collection
	}

	// Create the client and connect
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		log.Printf("Failed to create the MongoDB client: %v", err)
		return err
	}

	// Connect to the database
	ctx, cancel := TimedContext()
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Printf("Could not connect to MongoDB: %v", err)
		return err
	}

	// Setup a hook on SIGTERM/SIGINT and disconnect the client before exiting
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Disconnecting MongoDB.")
		ctx, _ := TimedContext()
		client.Disconnect(ctx)
		os.Exit(1)
	}()

	// Ping the database to make sure it's available
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("Could not ping MongoDB: %v", err)
		client.Disconnect(ctx)
		return err
	}

	// Set the shared collection
	db := client.Database(db_name)
	coll = db.Collection(coll_name)

	// Define a unique index on key `short`
	_, err = coll.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bson.D{{Key: "short", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		log.Printf("Could not create index: %v", err)
		client.Disconnect(ctx)
		return err
	}

	// Successfully connected to the database
	log.Println("Connected to MongoDB!")
	return nil
}

/* ****************************************** *\
 * *********** DATABASE FUNCTIONS *********** *
\* ****************************************** */

// GetAllShortlinks retrives all shortlinks from the db
func GetAllShortlinks() ([]*Shortlink, error) {
	ctx, cancel := TimedContext()
	defer cancel()

	// Find all documents in the collection
	cursor, err := coll.Find(ctx, bson.D{})
	if err != nil {
		log.Printf("Error receiving all shortlinks: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// Unmarshal via cursor.All
	var shortlinks []*Shortlink = []*Shortlink{}
	err = cursor.All(ctx, &shortlinks)
	if err != nil {
		log.Printf("Error unmarshalling all: %v", err)
		return nil, err
	}

	return shortlinks, nil
}

// GetShortlinkByShort retrives a shortlink by its short from the database
func GetShortlinkByShort(short string) (*Shortlink, error) {
	ctx, cancel := TimedContext()
	defer cancel()

	// Filter based on the provided short
	filter := bson.M{"short": short}
	var shortlink *Shortlink
	err := coll.FindOne(ctx, filter).Decode(&shortlink)

	if err != nil {
		log.Printf("Failed finding shortlink: %v", err)
		return nil, err
	}

	return shortlink, nil
}

//Create a shortlink in the database
func Create(shortlink *Shortlink) error {

	shortlink.ID = primitive.NewObjectID()
	shortlink.CreatedAt = time.Now()
	shortlink.UpdatedAt = time.Now()

	ctx, cancel := TimedContext()
	defer cancel()

	_, err := coll.InsertOne(ctx, shortlink)

	if err != nil {
		log.Printf("Error creating shortlink: %v", err)
		return err
	}

	return nil
}

//Update an existing shortlink `short` in the database with new data `shortlink`
func Update(short string, shortlink *ShortlinkUpdate) (*Shortlink, error) {

	filter := bson.M{"short": short}

	shortlink.UpdatedAt = time.Now()
	update := bson.M{"$set": shortlink}

	opt := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(false)

	ctx, cancel := TimedContext()
	defer cancel()

	var updatedShortlink *Shortlink
	err := coll.FindOneAndUpdate(ctx, filter, update, opt).Decode(&updatedShortlink)
	if err != nil {
		log.Printf("Error updating shortlink: %v", err)
		return nil, err
	}
	return updatedShortlink, nil
}

//Delete an existing shortlink from the database
func Delete(short string) (int64, error) {

	filter := bson.M{"short": short}

	ctx, cancel := TimedContext()
	defer cancel()

	res, err := coll.DeleteMany(ctx, filter)

	if err != nil {
		log.Printf("Unexpected error deleting shortlink: %v", err)
		return -1, err
	}

	return res.DeletedCount, nil
}

// GetRedirect Retrives the URL for a shortlink from the database
func GetRedirect(short string) (string, error) {

	filter := bson.D{primitive.E{Key: "short", Value: short}}

	ctx, cancel := TimedContext()
	defer cancel()

	update := bson.D{primitive.E{Key: "$inc", Value: bson.D{primitive.E{Key: "access_count", Value: 1}}}}

	opt := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(false).SetProjection(bson.M{"long": 1})

	var result Shortlink
	err := coll.FindOneAndUpdate(ctx, filter, update, opt).Decode(&result)
	if err != nil {
		log.Printf("Error redirecting: %v", err)
		return "", err
	}

	return result.LongUrl, nil
}

// IsFree returns true if there is no matching shortlink in the database, false otherwise
func IsFree(short string) (bool, error) {

	filter := bson.D{primitive.E{Key: "short", Value: short}}
	ctx, cancel := TimedContext()
	defer cancel()

	err := coll.FindOne(ctx, filter).Err()
	if err != nil {
		if isNotFundError(err) {
			return true, nil
		}
		log.Printf("Unexpected error checking for free: %v", err)
		return false, err
	}

	return false, nil
}

/* ****************************************** *\
 * ************ HELPER FUNCTIONS ************ *
\* ****************************************** */

// TimedContext returns a timed context using the default timeout.
func TimedContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
}

// Unbound context for long operations
func UnboundContext() context.Context {
	return context.Background()
}

// Check if is Mongo no document fund error
func isNotFundError(err error) bool {
	return err == mongo.ErrNoDocuments
}

// Check if is duplicate error
func isDuplicateError(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
