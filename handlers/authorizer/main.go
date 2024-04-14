package main

import (
	"context"
	"fmt"
	"sam-app/game"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) events.APIGatewayCustomAuthorizerResponse {
	cookie := req.Headers["Cookie"]

	if strings.Contains(cookie, "GreatKingdomAuth") {
		payload, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomAuth=")+17:], ";")
		if strings.Compare(payload[:6], "bearer") == 0 {
			_, token, _ := strings.Cut(payload, " ")
			claim := game.AuthTokenClaims{}
			t, err := jwt.ParseWithClaims(token, &claim, func(t *jwt.Token) (interface{}, error) {
				return []byte("key"), nil
			})
			if err != nil {
				return events.APIGatewayCustomAuthorizerResponse{}
			}
			fmt.Printf("%+v %v", claim, t.Valid)

		} else {
			fmt.Print("bearer error")
		}
	}
	return events.APIGatewayCustomAuthorizerResponse{}
}

func main() {
	lambda.Start(handler)
}
