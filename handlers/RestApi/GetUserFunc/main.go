package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userItem, err := ddb.NewUserRepository().Get(ctx, req.QueryStringParameters["UserId"])
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	body, _ := json.Marshal(userItem)
	return auth.RESTResponse(200, auth.CORSHeaders, string(body)), nil
}

func main() {
	lambda.Start(handler)
}
