package main

import (
	"context"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	awsutils.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.PONGBROADCAST})
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
