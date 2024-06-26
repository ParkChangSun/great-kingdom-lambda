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

	body := struct {
		GameSessionName string
	}{}
	err = json.Unmarshal([]byte(req.Body), &body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	id := uuid.New().String()
	item, _ := attributevalue.MarshalMap(game.GameSessionDDBItem{
		GameSessionId:   id,
		GameSessionName: body.GameSessionName,
	})
	_, err = dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Item:      item,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	b, _ := json.Marshal(struct{ GameSessionId string }{GameSessionId: id})
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    game.DefaultCORSHeaders,
		Body:       string(b),
	}, nil
}

func main() {
	lambda.Start(handler)
}
