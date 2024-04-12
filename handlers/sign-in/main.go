package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sam-app/game"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &body)

	item, err := game.GetUser(ctx, body.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	if item.UserId == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "id not found", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(item.PasswordHash), []byte(body.Password))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "incorrect password", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	type AuthTokenClaims struct {
		jwt.RegisteredClaims
		UserId    string
		Timestamp int64
	}
	t := AuthTokenClaims{UserId: item.UserId, Timestamp: time.Now().UnixMilli()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, t)
	tstr, err := token.SignedString([]byte("key"))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Set-Cookie":                       fmt.Sprint("GreatKingdomAuth=bearer ", tstr, ";"),
			"Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     "GET, POST",
			"Access-Control-Allow-Headers":     "Content-Type, *",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
