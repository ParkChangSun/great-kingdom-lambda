package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sam-app/game"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cookie := req.Headers["Cookie"]
	if !strings.Contains(cookie, "GreatKingdomRefresh") {
		return events.APIGatewayProxyResponse{}, nil
	}
	payload, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomRefresh=")+20:], ";")
	refreshTokenStr, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	refreshToken := game.RefreshToken{}
	json.Unmarshal(refreshTokenStr, &refreshToken)

	userItem, err := game.GetUser(ctx, req.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if userItem.RefreshToken != string(refreshTokenStr) || refreshToken.Time.Before(time.Now()) {
		userItem.RefreshToken = "logout"
		userItem.UpdateRefreshToken(ctx)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Headers: map[string]string{
				// "Access-Control-Allow-Origin":      "http://localhost:5173",
				"Access-Control-Allow-Credentials": "true",
			},
		}, nil
	}

	signedToken, err := game.NewAuthToken(userItem.UserId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = game.NewRefreshToken(userItem.UserId)
	err = userItem.UpdateRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			// "Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
		},
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
