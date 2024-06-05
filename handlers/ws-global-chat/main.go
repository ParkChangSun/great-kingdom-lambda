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
	chat := struct {
		Chat      string
		ChatName  string
		Timestamp int64
	}{
		Chat:      fmt.Sprint(now.Format(time.TimeOnly), sender.UserId, data.Chat),
		ChatName:  "globalchat",
		Timestamp: now.UnixMicro(),
	}

	item, _ := attributevalue.MarshalMap(chat)
	_, err = dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("GLOBAL_CHAT_DYNAMODB")),
		Item:      item,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	b, _ := json.Marshal(chat)
	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
	})
	for _, v := range receivers {
		wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(v.ConnectionId),
			Data:         b,
		})
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
