package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, _ := config.LoadDefaultConfig(ctx)

	data := struct {
		Chat string
	}{}
	err := json.Unmarshal([]byte(req.Body), &data)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sender, err := game.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	key := expression.KeyEqual(expression.Key("GameSessionId"), expression.Value("globalchat"))
	expr, _ := expression.NewBuilder().WithKeyCondition(key).Build()
	out, err := dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		IndexName:                 aws.String("globalchat"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	receivers := []game.ConnectionDDBItem{}
	err = attributevalue.UnmarshalListOfMaps(out.Items, &receivers)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	now := time.Now()

	// this goes to connect route
	chatkey := expression.KeyEqual(expression.Key("ChatName"), expression.Value("globalchat"))
	cexpr, _ := expression.NewBuilder().WithKeyCondition(chatkey).Build()
	dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
		TableName: aws.String(os.Getenv("WEBSOCKET_ENDPOINT")),
		// ScanIndexForward: aws.Bool(true),
		KeyConditionExpression:    cexpr.KeyCondition(),
		ExpressionAttributeNames:  cexpr.Names(),
		ExpressionAttributeValues: cexpr.Values(),
	})

	chat, _ := json.Marshal(struct{ Chat string }{fmt.Sprint(now.Format(time.TimeOnly), sender.UserId, data.Chat)})
	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
	})
	for _, v := range receivers {
		wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(v.ConnectionId),
			Data:         chat,
		})
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
