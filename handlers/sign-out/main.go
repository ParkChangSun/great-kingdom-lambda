package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	st := struct{ UserId string }{}
	json.Unmarshal([]byte(req.Body), &st)
	userItem, _ := game.GetUser(ctx, st.UserId)
	userItem.RefreshToken = "logout"
	userItem.UpdateRefreshToken(ctx)

	authCookie := &http.Cookie{
		Name:     "GreatKingdomAuth",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		HttpOnly: true,
	}
	refreshCookie := &http.Cookie{
		Name:     "GreatKingdomAuth",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		HttpOnly: true,
	}
	return events.APIGatewayProxyResponse{StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
		},
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				authCookie.String(),
				refreshCookie.String(),
			},
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
