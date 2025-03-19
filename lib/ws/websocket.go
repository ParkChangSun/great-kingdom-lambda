package ws

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

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

func DeleteWebSocket(ctx context.Context, connectionId string) error {
	_, err := wsClient(ctx).DeleteConnection(ctx, &apigatewaymanagementapi.DeleteConnectionInput{
		ConnectionId: aws.String(connectionId),
	})
	return err
}
