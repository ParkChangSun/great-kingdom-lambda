package main

import (
	"context"
	"encoding/json"
	"os"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqBody := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	userItem, err := game.GetUser(ctx, reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
			},
			Body: err.Error(),
		}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(userItem.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
			},
			Body: err.Error(),
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

	resBody, _ := json.Marshal(struct{ Id string }{Id: userItem.UserId})
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    game.DefaultCORSHeaders,
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				game.GetCookieHeader("GreatKingdomAuth", signedToken, time.Now().Add(game.AUTHEXPIRES)),
				game.GetCookieHeader("GreatKingdomRefresh", userItem.RefreshToken, time.Now().Add(game.REFRESHEXPIRES)),
			},
		},
		Body: string(resBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}
