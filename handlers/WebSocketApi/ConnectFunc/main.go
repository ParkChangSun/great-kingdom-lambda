package main

import (
	"context"
	"sam-app/auth"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if _, ok := req.QueryStringParameters["GameTableId"]; !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    auth.CORSHeaders,
			Body:       "id not found",
		}, nil
	}

	err := ddb.PutConnInPool(ctx, ddb.ConnectionDDBItem{
		ConnectionId: req.RequestContext.ConnectionID,
		GameTableId:  req.QueryStringParameters["GameTableId"],
		UserId:       "",
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
