package main

import (
	"context"
	"encoding/json"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/game"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	move := game.Move{}
	err := json.Unmarshal([]byte(req.Body), &move)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	conn, err := ddb.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	r := ddb.Record{
		EventType:         vars.GAMEEVENT,
		ConnectionDDBItem: conn,
		Move:              move,
		Timestamp:         req.RequestContext.RequestTimeEpoch,
	}
	err = awsutils.SendToQueue(ctx, r, r.GameSessionId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
