package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if _, ok := req.QueryStringParameters["GameSessionId"]; !ok {
		return events.APIGatewayProxyResponse{}, err
	}

	if req.QueryStringParameters["GameSessionId"] == "globalchat" {
		item, _ := attributevalue.MarshalMap(game.ConnectionDDBItem{
			ConnectionId:  req.RequestContext.ConnectionID,
			Timestamp:     req.RequestContext.RequestTimeEpoch,
			GameSessionId: req.QueryStringParameters["GameSessionId"],
			UserId:        req.RequestContext.Authorizer.(map[string]interface{})["UserId"].(string),
		})
		dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")), Item: item})

		chatkey := expression.KeyEqual(expression.Key("ChatName"), expression.Value("globalchat"))
		expr, _ := expression.NewBuilder().WithKeyCondition(chatkey).Build()
		out, err := dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
			TableName: aws.String(os.Getenv("GLOBAL_CHAT_DYNAMODB")),
			// ScanIndexForward: aws.Bool(true),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
		})
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		lastmsgs := []struct {
			Chat      string
			ChatName  string
			Timestamp int64
		}{}
		attributevalue.UnmarshalListOfMaps(out.Items, &lastmsgs)

		log.Print("lastmsgs", lastmsgs)

		wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
			o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
		})
		for _, v := range lastmsgs {
			wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: aws.String(req.RequestContext.ConnectionID),
				Data:         []byte(v.Chat),
			})
		}

		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	msgbody, _ := json.Marshal(game.ConnectionDDBItem{
		ConnectionId:  req.RequestContext.ConnectionID,
		Timestamp:     req.RequestContext.RequestTimeEpoch,
		GameSessionId: req.QueryStringParameters["GameSessionId"],
		UserId:        req.RequestContext.Authorizer.(map[string]interface{})["UserId"].(string),
	})

	sqsClient := sqs.NewFromConfig(cfg)
	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
		MessageBody:    aws.String(string(msgbody)),
		MessageGroupId: aws.String(req.QueryStringParameters["GameSessionId"]),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"EventType": {DataType: aws.String("String"), StringValue: aws.String(game.JOINEVENT)},
		},
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
