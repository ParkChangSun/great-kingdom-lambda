package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	log.Print(req)

	// gameSessionId := "default"
	// if a, b := req.QueryStringParameters["GameSessionId"]; b {
	// 	gameSessionId = a
	// }
	dbItem, _ := attributevalue.MarshalMap(struct {
		ConnectionId string
	}{req.RequestContext.ConnectionID})

	out, err := dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:    aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Key:          dbItem,
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	outobj := game.WebSocketClient{}
	attributevalue.UnmarshalMap(out.Attributes, &outobj)

	msgbody, err := json.Marshal(chat.Chat{
		Timestamp:     time.Now().UnixMilli(),
		ConnectionId:  req.RequestContext.ConnectionID,
		GameSessionId: outobj.GameSessionId,
		Message:       fmt.Sprintf("%s has left.", req.RequestContext.ConnectionID),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sqsClient := sqs.NewFromConfig(cfg)

	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
		MessageBody:    aws.String(string(msgbody)),
		MessageGroupId: aws.String(outobj.GameSessionId),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
