package main

import (
	"context"
	"encoding/json"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userItem, err := game.GetUser(ctx, req.QueryStringParameters["UserId"])
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	body, _ := json.Marshal(userItem)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    game.DefaultCORSHeaders,
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}
