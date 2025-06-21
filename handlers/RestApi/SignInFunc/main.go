package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqBody := auth.Authenticate{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	user, err := ddb.NewUserRepository().Get(ctx, reqBody.Id)
	if err != nil {
		b, _ := json.Marshal(auth.ErrorResponseBody{Message: "login failed"})
		return auth.RESTResponse(400, auth.CORSHeaders, string(b)), nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		b, _ := json.Marshal(auth.ErrorResponseBody{Message: "login failed"})
		return auth.RESTResponse(400, auth.CORSHeaders, string(b)), nil
	}

	a, r, err := auth.GenerateTokenSet(reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	user.RefreshToken = r
	err = ddb.NewUserRepository().Put(ctx, user)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	resBody, _ := json.Marshal(auth.AuthBody{Authorized: true, AccessToken: a, Id: user.Id})
	return auth.RESTResponse(200, auth.AuthHeaders(r), string(resBody)), nil
}

func main() {
	lambda.Start(handler)
}
