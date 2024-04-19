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

	t := game.AuthTokenClaims{
		UserId: item.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(time.Minute * 15),
			},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, t)
	tstr, err := token.SignedString([]byte("key"))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	idbody, _ := json.Marshal(struct{ Id string }{Id: body.Id})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Set-Cookie": fmt.Sprint(
				"GreatKingdomAuth=bearer ",
				tstr,
				";domain=execute-api.us-east-1.amazonaws.com;path=/;Max-Age=900;HttpOnly;SameSite=None;Secure",
			),
			"Access-Control-Allow-Origin":      "http://localhost:5173",
			"Access-Control-Allow-Credentials": "true",
		},
		Body: string(idbody),
	}, nil
}

func main() {
	lambda.Start(handler)
}
