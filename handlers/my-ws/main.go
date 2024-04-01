package main

import (
	"context"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sqsClient := sqs.NewFromConfig(cfg)
	_ = sqs.NewListQueuesPaginator(sqsClient, &sqs.ListQueuesInput{})
	dbClient := dynamodb.NewFromConfig(cfg)

	tableName := aws.String(os.Getenv("STATEDYNAMODB"))
	t, _ := attributevalue.MarshalMap(game.WebSocketClient{Id: req.RequestContext.ConnectionID})
	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           tableName,
		Item:                t,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
