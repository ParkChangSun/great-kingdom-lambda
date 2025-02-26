package awsutils

import (
	"context"
	"encoding/json"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var _sqsClient *sqs.Client

func sqsClient(ctx context.Context) *sqs.Client {
	if _sqsClient == nil {
		cfg, _ := config.LoadDefaultConfig(ctx)
		_sqsClient = sqs.NewFromConfig(cfg)
	}
	return _sqsClient
}

func SendToQueue(ctx context.Context, record interface{}, groupId string) error {
	body, _ := json.Marshal(record)

	_, err := sqsClient(ctx).SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(vars.WEBSOCKET_EVENT_QUEUE),
		MessageBody:    aws.String(string(body)),
		MessageGroupId: aws.String(groupId),
	})

	return err
}

var _wsClient *apigatewaymanagementapi.Client

func wsClient(ctx context.Context) *apigatewaymanagementapi.Client {
	if _wsClient == nil {
		cfg, _ := config.LoadDefaultConfig(ctx)
		_wsClient = apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
			o.BaseEndpoint = aws.String(vars.WEBSOCKET_ENDPOINT)
		})
	}
	return _wsClient

}

func SendWebsocketMessage(ctx context.Context, connectionId string, payload any) error {
	data, _ := json.Marshal(payload)
	_, err := wsClient(ctx).PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionId),
		Data:         data,
	})
	return err
}

func RESTResponse(statusCode int, headers map[string]string, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}

type RESTErrorMessage struct {
	Message string `json:"message"`
}
