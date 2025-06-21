package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var _client *dynamodb.Client

func init() {
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	_client = dynamodb.NewFromConfig(cfg)
}

type ItemNotFoundError struct {
	TableName string
	Key       string
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("gk:ddb - item not found in table '%s' with key '%s'", e.TableName, e.Key)
}
