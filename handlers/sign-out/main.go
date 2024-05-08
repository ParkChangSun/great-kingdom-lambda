package main

import (
	"context"
	"encoding/json"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	st := struct{ UserId string }{}
	err := json.Unmarshal([]byte(req.Body), &st)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	userItem, err := game.GetUser(ctx, st.UserId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = "logout"
	userItem.UpdateRefreshToken(ctx)

	return events.APIGatewayProxyResponse{StatusCode: 200,
		Headers: map[string]string{
			// "Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
		},
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				game.GetCookieHeader("GreatKingdomAuth", "logout", time.Now().Add(game.EXPIRED)),
				game.GetCookieHeader("GreatKingdomRefresh", "logout", time.Now().Add(game.EXPIRED)),
			},
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
