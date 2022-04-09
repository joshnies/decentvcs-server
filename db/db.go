package db

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Global MongoDB client instance
var I *mongo.Client
var DB *mongo.Database

// Create MongoDB client
func Init() {
	I, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("DB_URI")))
	if err != nil {
		panic(err)
	}

	DB = I.Database(os.Getenv("DB_NAME"))
}
