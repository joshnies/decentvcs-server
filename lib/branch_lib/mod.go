package branch_lib

import (
	"context"
	"time"

	"github.com/joshnies/decent-vcs/config"
	"github.com/joshnies/decent-vcs/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Get branch with its latest commit using a MongoDB aggregation pipeline.
//
// @param pid - project ID
//
// @param bid - branch ID
//
func GetBranchWithCommit(pid primitive.ObjectID, bid primitive.ObjectID) (*models.BranchWithCommit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, []bson.M{
		{
			"$match": bson.M{
				"project_id": pid,
				"deleted_at": bson.M{"$exists": false},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "commits",
				"localField":   "commit_id",
				"foreignField": "_id",
				"as":           "commit",
			},
		},
		{
			"$unwind": "$commit",
		},
		{
			"$unset": "commit_id",
		},
		{
			"$sort": bson.M{
				"commit.index": -1,
			},
		},
		{
			"$limit": 1,
		},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Decode first branch
	cur.Next(ctx)
	var branch models.BranchWithCommit
	err = cur.Decode(&branch)
	if err != nil {
		return nil, err
	}

	return &branch, nil
}
