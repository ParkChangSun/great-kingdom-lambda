package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sugarlogger"
	"great-kingdom-lambda/lib/vars"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Print(ctx)
	sugar := sugarlogger.GetSugar()

	reqBody := struct{ Id string }{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	sugar.Info(reqBody.Id, "request delete user")

	err := ddb.NewUserRepository().Delete(ctx, reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sugar.Info(reqBody.Id, "delete success")

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
