package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sam-app/game"

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
		return events.APIGatewayProxyResponse{}, err
	}
	if userItem.UserId == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "id not found", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(userItem.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "incorrect password", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
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
		Body: string(resBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}
