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
	reqBody := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &reqBody)

	userItem, err := ddb.GetUser(ctx, reqBody.Id)
	if err != nil {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "user not found"), nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(userItem.PasswordHash), []byte(reqBody.Password))
	if err != nil {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "password incorrect"), nil
	}

	a, r, err := auth.GenerateTokenSet(reqBody.Id)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userItem.RefreshToken = r
	err = userItem.SyncRefreshToken(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	resBody, _ := json.Marshal(struct{ Id string }{Id: userItem.UserId})
	return awsutils.RESTResponse(200, auth.AuthHeaders(a, r), string(resBody)), nil
}

func main() {
	lambda.Start(handler)
}
