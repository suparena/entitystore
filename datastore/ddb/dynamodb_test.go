/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-openapi/strfmt"
	"github.com/joho/godotenv"
	"github.com/suparena/entitystore/datastore"
	"github.com/suparena/entitystore/datastore/testmodels"
	"log"
	"os"
	"testing"
	"time"
)

func getRatingSystemStore() (datastore.DataStore[testmodels.RatingSystem], error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, proceeding with environment variables")
		return nil, err
	}

	awsAccessKey := os.Getenv("AWS_ACCESS_KEY")
	awsSecretKey := os.Getenv("AWS_SECRET_KEY")
	awsDDBTableName := os.Getenv("AWS_DDB_TABLE")

	// Get the AWS region from environment variables
	region := os.Getenv("AWS_REGION")

	storage, err := NewDynamodbDataStore[testmodels.RatingSystem](awsAccessKey, awsSecretKey, region, awsDDBTableName)
	return storage, err
}

func TestNewDynamoDBStoragePut(t *testing.T) {
	storage, err := getRatingSystemStore()
	if err != nil {
		t.Error(err)
	}

	ct := strfmt.DateTime(time.Now())
	ratingSystem := &testmodels.RatingSystem{
		ID:          aws.String("TTOakville"),
		Name:        aws.String("Oalville Table Tennis Ranking System (test)"),
		Description: aws.String("This is a test rating system for Oakville Table Tennis Club"),
		CreatedAt:   &ct,
		UpdatedAt:   &ct,
	}

	err = storage.Put(context.Background(), *ratingSystem)
	if err != nil {
		t.Error(err)
	}
}

func TestNewDynamoDBStorageGetOne(t *testing.T) {
	storage, err := getRatingSystemStore()
	if err != nil {
		t.Error(err)
	}

	rs, err := storage.GetOne(context.Background(), "TTOakville")
	if err != nil {
		t.Error(err)
	}

	t.Logf("Rating System: %v", rs)
}

func TestNewDynamoDBStorageDelete(t *testing.T) {
	storage, err := getRatingSystemStore()
	if err != nil {
		t.Error(err)
	}

	err = storage.Delete(context.Background(), "TTOakville")
	if err != nil {
		t.Error(err)
	}

	t.Logf("Rating System deleted")
}
