package ddb

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var ddbClient *dynamodb.Client

func client(ctx context.Context) *dynamodb.Client {
	if ddbClient == nil {
		cfg, _ := config.LoadDefaultConfig(ctx)
		ddbClient = dynamodb.NewFromConfig(cfg)
	}
	return ddbClient
}

var ErrItemNotFound = errors.New("dynamodb item not found")
