package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sam-app/chat"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// save connectionid in the game session?

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	gameSessionId := "default"
	if a, b := req.QueryStringParameters["GameSessionId"]; b {
		gameSessionId = a
	}
	dbItem, _ := attributevalue.MarshalMap(game.WebSocketClient{
		ConnectionId:  req.RequestContext.ConnectionID,
		GameSessionId: gameSessionId,
		UserId:        "header-required",
	})
	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Item:                dbItem,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	msgbody, err := json.Marshal(chat.Chat{
		Timestamp:     time.Now().UnixMilli(),
		ConnectionId:  req.RequestContext.ConnectionID,
		GameSessionId: gameSessionId,
		Message:       fmt.Sprintf("%s has joined.", req.RequestContext.ConnectionID),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sqsClient := sqs.NewFromConfig(cfg)

	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
		MessageBody:    aws.String(string(msgbody)),
		MessageGroupId: aws.String(gameSessionId),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
