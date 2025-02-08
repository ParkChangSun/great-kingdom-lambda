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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
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
		c := ddb.ConnectionDDBItem{
			ConnectionId:  req.RequestContext.ConnectionID,
			Timestamp:     req.RequestContext.RequestTimeEpoch,
			GameSessionId: req.QueryStringParameters["GameSessionId"],
			UserId:        req.RequestContext.Authorizer.(map[string]interface{})["UserId"].(string),
		}

		item, _ := attributevalue.MarshalMap(c)
		dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")), Item: item})

		msgbody, _ := json.Marshal(c)
		_, err = sqs.NewFromConfig(cfg).SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:       aws.String(os.Getenv("POST_MESSAGE_QUEUE")),
			MessageBody:    aws.String(string(msgbody)),
			MessageGroupId: aws.String(req.QueryStringParameters["GameSessionId"]),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"EventType": {DataType: aws.String("String"), StringValue: aws.String(awsutils.GLOBALCHAT)},
			},
		})
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	msgbody, _ := json.Marshal(ddb.ConnectionDDBItem{
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
			"EventType": {DataType: aws.String("String"), StringValue: aws.String(awsutils.JOINEVENT)},
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
