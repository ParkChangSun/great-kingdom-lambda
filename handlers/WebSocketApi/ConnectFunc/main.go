package main

import (
	"context"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.QueryStringParameters["GameTableId"]

	if id == "globalchat" {
		err := ddb.PutConnInPool(ctx, ddb.ConnectionDDBItem{
			ConnectionId: req.RequestContext.ConnectionID,
			GameTableId:  "globalchat",
			UserId:       "",
		})
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
	}

	if id == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    auth.CORSHeaders,
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
