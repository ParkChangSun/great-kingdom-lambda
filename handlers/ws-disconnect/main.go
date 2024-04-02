package main

import (
	"context"
	"log"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	log.Print("remove ", req.RequestContext.ConnectionID)
	dbItem, err := attributevalue.MarshalMap(game.WebSocketClient{Id: req.RequestContext.ConnectionID})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	_, err = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("STATEDYNAMODB")),
		Key:       dbItem,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	// msgbody, err := json.Marshal(chat.Msg{
	// 	Timestamp: time.Now().UnixMilli(),
	// 	Id:        req.RequestContext.ConnectionID,
	// 	RoomId:    "default",
	// 	Chat:      "connected someone",
	// })
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }

	// sqsClient := sqs.NewFromConfig(cfg)

	// _, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
	// 	QueueUrl:       aws.String(os.Getenv("QUEUE")),
	// 	MessageBody:    aws.String(string(msgbody)),
	// 	MessageGroupId: aws.String("default"),
	// })
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
