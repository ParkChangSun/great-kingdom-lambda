package main

import (
	"context"
	"encoding/json"
	"errors"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"

	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userRepo := ddb.NewUserRepository()

	cookie := req.Headers["cookie"]
	if !strings.Contains(cookie, "GreatKingdomRefresh=") {
		body, _ := json.Marshal(auth.AuthBody{Authorized: false, AccessToken: "", Id: ""})
		return auth.RESTResponse(400, auth.AuthHeaders(""), string(body)), nil
	}

	refreshTokenStr, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomRefresh=")+20:], ";")
	refreshToken, refreshTokenClaims, err := auth.ParseToken(refreshTokenStr)

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return events.APIGatewayProxyResponse{}, err
	}

	user, err := userRepo.Get(ctx, refreshTokenClaims.Subject)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if refreshToken.Valid {
		a, r, err := auth.GenerateTokenSet(refreshTokenClaims.Subject)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		user.RefreshToken = r
		err = userRepo.Put(ctx, user)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		body, _ := json.Marshal(auth.AuthBody{Authorized: true, AccessToken: a, Id: user.Id})
		return auth.RESTResponse(200, auth.AuthHeaders(r), string(body)), nil
	}

	user.RefreshToken = ""
	err = userRepo.Put(ctx, user)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	body, _ := json.Marshal(auth.AuthBody{Authorized: false, AccessToken: "", Id: ""})
	return auth.RESTResponse(200, auth.AuthHeaders(""), string(body)), nil
}

func main() {
	lambda.Start(handler)
}
