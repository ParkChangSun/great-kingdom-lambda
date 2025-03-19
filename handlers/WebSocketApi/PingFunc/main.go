package main

import (
	"context"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	ws.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.PONGBROADCAST})
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
