package main

import (
	"context"
	"great-kingdom-lambda/lib/ddb"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	err := ddb.NewConnectionRepository().Put(ctx, ddb.Connection{
		Id:          req.RequestContext.ConnectionID,
		GameTableId: req.QueryStringParameters["GameTableId"],
		UserId:      req.QueryStringParameters["UserId"],
		CreatedDate: time.Now().Format(time.RFC3339),
		Authorized:  false,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
