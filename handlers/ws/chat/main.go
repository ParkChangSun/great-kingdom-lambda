package main

import (
	"context"
	"encoding/json"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	chat := struct{ Chat string }{}
	err := json.Unmarshal([]byte(req.Body), &chat)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	conn, err := ddb.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	r := ddb.Record{
		EventType:         vars.CHATEVENT,
		ConnectionDDBItem: conn,
		Chat:              chat.Chat,
		Timestamp:         req.RequestContext.RequestTimeEpoch,
	}
	err = awsutils.SendToQueue(ctx, r, r.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
