package main

import (
	"context"
	"os"
	"sam-app/auth"
	"sam-app/ddb"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userItem, err := ddb.GetUser(ctx, req.RequestContext.Authorizer["UserId"].(string))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = ""
	err = userItem.SyncRefreshToken(ctx)
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

func main() {
	lambda.Start(handler)
}
