package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqBody := struct {
		UserId       string
		RefreshToken string
	}{}
	err := json.Unmarshal([]byte(req.Body), &reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	userItem, err := game.GetUser(ctx, req.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	refreshTokenStr, err := base64.RawStdEncoding.DecodeString(reqBody.RefreshToken)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	refreshToken := game.RefreshToken{}
	err = json.Unmarshal(refreshTokenStr, &refreshToken)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if userItem.RefreshToken != reqBody.RefreshToken || refreshToken.Time.Before(time.Now()) {
		userItem.RefreshToken = "logout"
		userItem.UpdateRefreshToken(ctx)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":      "http://localhost:5173",
				"Access-Control-Allow-Credentials": "true",
			},
		}, nil
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
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
		},
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				fmt.Sprint(
					"GreatKingdomAuth=bearer ",
					signedToken,
					";domain=execute-api.us-east-1.amazonaws.com;path=/;Max-Age=900;HttpOnly;SameSite=None;Secure;Partitioned",
				),
				fmt.Sprint(
					"GreatKingdomRefresh=",
					userItem.RefreshToken,
					";domain=execute-api.us-east-1.amazonaws.com;path=/;Max-Age=3600;HttpOnly;SameSite=None;Secure;Partitioned",
				),
			},
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
