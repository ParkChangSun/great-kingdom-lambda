package main

import (
	"context"
	"encoding/json"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	body := struct {
		GameSessionName string
	}{}
	err = json.Unmarshal([]byte(req.Body), &body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	item, _ := attributevalue.MarshalMap(game.GameSession{
		GameSessionId:   uuid.New().String(),
		GameSessionName: req.Body,
	})
	dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Item:      item,
	})

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
