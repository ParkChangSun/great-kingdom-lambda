package main

import (
	"context"
	"encoding/json"
	"errors"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cookie := req.Headers["Cookie"]
	if !strings.Contains(cookie, "GreatKingdomRefresh=") {
		body, _ := json.Marshal(auth.AuthBody{Authorized: false, AccessToken: "", Id: ""})
		return awsutils.RESTResponse(400, auth.AuthHeaders(""), string(body)), nil
	}

	refreshTokenStr, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomRefresh=")+20:], ";")
	refreshTokenClaims := jwt.RegisteredClaims{}
	refreshToken, err := jwt.ParseWithClaims(refreshTokenStr, &refreshTokenClaims, func(t *jwt.Token) (any, error) {
		return []byte("key"), nil
	})

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return events.APIGatewayProxyResponse{}, err
	}

	user, err := ddb.GetUser(ctx, refreshTokenClaims.Subject)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if refreshToken.Valid {
		a, r, err := auth.GenerateTokenSet(refreshTokenClaims.Subject)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		user.RefreshToken = r
		err = user.SyncRefreshToken(ctx)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		body, _ := json.Marshal(auth.AuthBody{Authorized: true, AccessToken: a, Id: user.UserId})
		return awsutils.RESTResponse(200, auth.AuthHeaders(r), string(body)), nil
	}

	user.RefreshToken = ""
	err = user.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	body, _ := json.Marshal(auth.AuthBody{Authorized: false, AccessToken: "", Id: ""})
	return awsutils.RESTResponse(200, auth.AuthHeaders(""), string(body)), nil
}

func main() {
	lambda.Start(handler)
}
