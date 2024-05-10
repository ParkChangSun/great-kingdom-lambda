package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	refreshTokenStr := game.ParseRefreshToken(req.Headers["Cookie"])
	userItem, err := game.GetUserByRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return game.SignOutResponse, nil
	}

	refreshTokenDecode, err := base64.RawStdEncoding.DecodeString(refreshTokenStr)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	refreshToken := game.RefreshToken{}
	json.Unmarshal([]byte(refreshTokenDecode), &refreshToken)

	if refreshToken.Time.Before(time.Now()) {
		userItem.RefreshToken = "logout"
		userItem.UpdateRefreshToken(ctx)
		return game.SignOutResponse, nil
	}

	signedToken, err := game.NewAuthToken(userItem.UserId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = game.NewRefreshToken()
	err = userItem.UpdateRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    game.DefaultCORSHeaders,
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				game.GetCookieHeader("GreatKingdomAuth", signedToken, time.Now().Add(game.AUTHEXPIRES)),
				game.GetCookieHeader("GreatKingdomRefresh", userItem.RefreshToken, time.Now().Add(game.REFRESHEXPIRES)),
			},
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
