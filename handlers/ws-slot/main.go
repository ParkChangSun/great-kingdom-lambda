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
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, _ := config.LoadDefaultConfig(ctx)

	conn, err := game.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	msgbody, _ := json.Marshal(game.GameChatSQSRecord{
		Timestamp:     req.RequestContext.RequestTimeEpoch,
		ConnectionId:  conn.ConnectionId,
		UserId:        conn.UserId,
		GameSessionId: conn.GameSessionId,
	})

	sqsClient := sqs.NewFromConfig(cfg)
	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
		MessageBody:    aws.String(string(msgbody)),
		MessageGroupId: aws.String(conn.GameSessionId),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"EventType": {DataType: aws.String("String"), StringValue: aws.String(game.SLOTEVENT)},
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
