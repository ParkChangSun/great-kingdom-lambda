package main

import (
	"context"
	"fmt"
	"os"
	"sam-app/auth"
	"sam-app/ddb"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cookie := req.Headers["Cookie"]
	refreshTokenStr, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomRefresh=")+20:], ";")
	refreshTokenClaims := jwt.RegisteredClaims{}
	refreshToken, err := jwt.ParseWithClaims(refreshTokenStr, &refreshTokenClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte("key"), nil
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	u, err := ddb.GetUser(ctx, refreshTokenClaims.Subject)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if !refreshToken.Valid {
		u.RefreshToken = ""
		err = u.SyncRefreshToken(ctx)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
				"Set-Cookie":                       auth.CookieHeader("GreatKingdomRefresh", "", time.Now().Add(auth.EXPIRED)),
			},
		}, nil
	}

	if u.RefreshToken != refreshTokenStr {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("invalid logout")
	}

	newAccessToken, newRefreshToken, err := auth.GenerateTokenSet(refreshTokenClaims.Subject)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	u.RefreshToken = newRefreshToken
	err = u.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    auth.AuthHeaders(newAccessToken, newRefreshToken),
	}, nil
}

func main() {
	lambda.Start(handler)
}
