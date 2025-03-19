package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/vars"

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

	resBody, _ := json.Marshal(auth.AuthBody{Authorized: false, AccessToken: "", Id: ""})
	return auth.RESTResponse(200, map[string]string{
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Origin":      vars.CLIENT_ORIGIN,
		"Set-Cookie":                       auth.ExpiredCookie,
	}, string(resBody)), nil
}

func main() {
	lambda.Start(handler)
}
