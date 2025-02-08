package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sam-app/auth"
	"sam-app/ddb"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	refreshTokenStr := auth.ParseRefreshToken(req.Headers["Cookie"])
	userItem, err := ddb.GetUserByRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return auth.SignOutResponse, nil
	}

	refreshTokenDecode, err := base64.RawStdEncoding.DecodeString(refreshTokenStr)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	refreshToken := auth.RefreshToken{}
	json.Unmarshal([]byte(refreshTokenDecode), &refreshToken)

	if refreshToken.Time.Add(auth.REFRESHEXPIRES).Before(time.Now()) {
		userItem.RefreshToken = "logout"
		userItem.SyncRefreshToken(ctx)
		return auth.SignOutResponse, nil
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
