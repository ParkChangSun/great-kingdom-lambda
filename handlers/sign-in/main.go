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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqBody := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	item, err := game.GetUser(ctx, reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	if item.UserId == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "id not found", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(item.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "incorrect password", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, game.AuthTokenClaims{
		UserId: item.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(time.Minute * 15),
			},
		},
	})
	signedToken, err := token.SignedString([]byte("key"))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	refreshToken, _ := json.Marshal(struct {
		RefreshId string
		Time      time.Time
	}{
		uuid.NewString(),
		time.Now(),
	})
	refreshToken64 := base64.StdEncoding.EncodeToString(refreshToken)
	item.RefreshToken = refreshToken64
	err = item.UpdateRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	resBody, _ := json.Marshal(struct{ Id string }{Id: item.UserId})
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
					refreshToken64,
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
