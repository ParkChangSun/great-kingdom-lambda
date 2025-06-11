package ddb

import (
	"context"
	"fmt"

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

type ItemNotFoundError struct {
	TableName string
	Key       string
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("gk:ddb - item not found in table '%s' with key '%s'", e.TableName, e.Key)
}
