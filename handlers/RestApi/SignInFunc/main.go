package main

import (
	"context"
	"encoding/json"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqBody := auth.Authenticate{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	user, err := ddb.GetUser(ctx, reqBody.Id)
	if err != nil {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "login failed"), nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "login failed"), nil
	}

	a, r, err := auth.GenerateTokenSet(reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	user.RefreshToken = r
	err = user.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	resBody, _ := json.Marshal(auth.AuthBody{Authorized: true, AccessToken: a, Id: user.UserId})
	return awsutils.RESTResponse(200, auth.AuthHeaders(r), string(resBody)), nil
}

func main() {
	lambda.Start(handler)
}
