package main

import (
	"context"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	refreshTokenStr := game.ParseRefreshToken(req.Headers["Cookie"])
	userItem, err := game.GetUserByRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return game.SignOutResponse, nil
	}

	userItem.RefreshToken = "logout"
	err = userItem.UpdateRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return game.SignOutResponse, nil
}

func main() {
	lambda.Start(handler)
}
