package main

import (
	"context"
	"encoding/json"
	"os"
	"sam-app/awsutils"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	// key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{req.RequestContext.ConnectionID})
	// out, err := dynamodb.NewFromConfig(cfg).DeleteItem(ctx, &dynamodb.DeleteItemInput{
	// 	TableName:    aws.String(os.Getenv("CONNECTION_DYNAMODB")),
	// 	Key:          key,
	// 	ReturnValues: "ALL_OLD",
	// })
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }

	// item := game.ConnectionDDBItem{}
	// err = attributevalue.UnmarshalMap(out.Attributes, &item)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }

	conn, err := ddb.DeleteConnInPool(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if conn.GameSessionId == "globalchat" {
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	conn.Timestamp = req.RequestContext.RequestTimeEpoch
	msgbody, _ := json.Marshal(conn)
	_, err = sqs.NewFromConfig(cfg).SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
		MessageBody:    aws.String(string(msgbody)),
		MessageGroupId: aws.String(conn.GameSessionId),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"EventType": {DataType: aws.String("String"), StringValue: aws.String(awsutils.LEAVEEVENT)},
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
