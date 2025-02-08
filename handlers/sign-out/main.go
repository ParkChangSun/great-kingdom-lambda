package main

import (
	"context"
	"sam-app/auth"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	refreshTokenStr := auth.ParseRefreshToken(req.Headers["Cookie"])
	userItem, err := ddb.GetUserByRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return auth.SignOutResponse, nil
	}

	userItem.RefreshToken = "logout"
	err = userItem.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return auth.SignOutResponse, nil
}

func main() {
	lambda.Start(handler)
}
