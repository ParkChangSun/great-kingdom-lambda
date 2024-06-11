package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) {
	cfg, _ := config.LoadDefaultConfig(ctx)

	pong, _ := json.Marshal(struct{ EventType string }{EventType: "pong"})
	apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
	}).PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(req.RequestContext.ConnectionID),
		Data:         pong,
	})
}

func main() {
	lambda.Start(handler)
}
