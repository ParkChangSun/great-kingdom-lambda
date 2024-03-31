package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var wsClient *apigatewaymanagementapi.Client
var dbClient *dynamodb.Client

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Print("connected", req.RequestContext.ConnectionID)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	wsClient = apigatewaymanagementapi.NewFromConfig(cfg)
	dbClient = dynamodb.NewFromConfig(cfg)

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
	log.Print("connection id saved")

	_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(req.RequestContext.ConnectionID),
		Data:         []byte(fmt.Sprintf("%s connected", req.RequestContext.ConnectionID)),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	log.Print("message sent")

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
