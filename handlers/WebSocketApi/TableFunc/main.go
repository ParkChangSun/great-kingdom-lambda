package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	e := sqs.GameTableEvent{}
	err := json.Unmarshal([]byte(req.Body), &e)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	conn, err := ddb.NewConnectionRepository().Get(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	record := sqs.Record{
		GameTableEvent: e,
		Connection:     conn,
		Timestamp:      req.RequestContext.RequestTimeEpoch,
	}
	err = sqs.SendToQueue(ctx, record, record.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
