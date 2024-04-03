package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sam-app/chat"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func handler(ctx context.Context, req events.SQSEvent) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	payload := chat.Chat{}
	err = json.Unmarshal([]byte(req.Records[0].Body), &payload)
	if err != nil {
		return err
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	k := expression.Key("GameSessionId").Equal(expression.Value(payload.GameSessionId))
	expr, err := expression.NewBuilder().WithKeyCondition(k).Build()
	if err != nil {
		return err
	}

	queryOutput, err := dbClient.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		IndexName:                 aws.String("ByGameSessionId"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return err
	}

	connections := []game.WebSocketClient{}
	err = attributevalue.UnmarshalListOfMaps(queryOutput.Items, connections)
	if err != nil {
		return err
	}

	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("API_ENDPOINT"))
	})

	for _, v := range connections {
		_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(v.ConnectionId),
			Data:         []byte(payload.Message),
		})
		if err != nil {
			log.Print(err)
		}
		// goneexception drop dynamodb item
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
