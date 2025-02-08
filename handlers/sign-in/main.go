package main

import (
	"context"
	"encoding/json"
	"os"
	"sam-app/auth"
	"sam-app/ddb"
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

	userItem, err := ddb.GetUser(ctx, reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
			},
			Body: "id not found",
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
			Body: "password incorrect",
		}, nil
	}

	signedToken, err := auth.NewAuthToken(userItem.UserId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = auth.NewRefreshToken()
	err = userItem.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	resBody, _ := json.Marshal(struct{ Id string }{Id: userItem.UserId})
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    auth.DefaultCORSHeaders,
		MultiValueHeaders: map[string][]string{
			"Set-Cookie": {
				auth.GetCookieHeader("GreatKingdomAuth", signedToken, time.Now().Add(auth.AUTHEXPIRES)),
				auth.GetCookieHeader("GreatKingdomRefresh", userItem.RefreshToken, time.Now().Add(auth.REFRESHEXPIRES)),
			},
		},
		Body: string(resBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}
