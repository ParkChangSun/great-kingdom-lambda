package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) error {
	cfg, _ := config.LoadDefaultConfig(ctx)

	body := struct{ LastScroll int64 }{}
	json.Unmarshal([]byte(req.Body), &body)

	if body.LastScroll == 0 {
		return nil
	}

	start, _ := attributevalue.MarshalMap(struct {
		ChatName  string
		Timestamp int64
	}{
		ChatName:  "globalchat",
		Timestamp: body.LastScroll,
	})

	chatkey := expression.KeyEqual(expression.Key("ChatName"), expression.Value("globalchat"))
	expr, _ := expression.NewBuilder().WithKeyCondition(chatkey).Build()
	out, err := dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("GLOBAL_CHAT_DYNAMODB")),
		ScanIndexForward:          aws.Bool(false),
		Limit:                     aws.Int32(50),
		ExclusiveStartKey:         start,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return err
	}

	lastKey := struct{ Timestamp int64 }{}
	attributevalue.UnmarshalMap(out.LastEvaluatedKey, &lastKey)
	lastmsgs := []struct {
		Chat      string
		ChatName  string
		Timestamp int64
	}{}
	attributevalue.UnmarshalListOfMaps(out.Items, &lastmsgs)
	payload := struct {
		EventType string
		Messages  []struct {
			Chat      string
			ChatName  string
			Timestamp int64
		}
		LastScroll int64
	}{
		EventType:  "scroll",
		Messages:   lastmsgs,
		LastScroll: lastKey.Timestamp,
	}
	b, _ := json.Marshal(payload)

	_, err = apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
	}).PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(req.RequestContext.ConnectionID),
		Data:         b,
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
