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
func GetOneWithCommit(teamID primitive.ObjectID, projectName string, branchName string) (*models.BranchWithCommit, error) {
	// TODO: Use project and branch ObjectIDs instead of names to allow for external fetching

	// Get project from database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project
	if err := config.MI.DB.Collection("projects").FindOne(ctx, bson.M{"team_id": teamID, "name": projectName}).Decode(&project); err != nil {
		return nil, err
	}

	// Get branch from database
	cur, err := config.MI.DB.Collection("branches").Aggregate(ctx, []bson.M{
		{
			"$match": bson.M{
				"deleted_at": bson.M{"$exists": false},
				"project_id": project.ID,
				"name":       branchName,
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
