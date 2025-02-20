package main

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &body)

	_, err := ddb.GetUser(ctx, body.Id)
	if !errors.Is(err, ddb.ErrItemNotFound) {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "id exists"), nil
	}

	idlen := regexp.MustCompile(`^[0-9a-zA-Z]{4,30}$`)
	if !idlen.Match([]byte(body.Id)) {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "invalid id"), nil
	}

	num := regexp.MustCompile(`[0-9]`)
	eng := regexp.MustCompile(`[a-zA-Z]`)
	bytelen := regexp.MustCompile(`^[a-zA-Z0-9@#$%^&*]{4,30}$`)
	if !num.Match([]byte(body.Password)) || !eng.Match([]byte(body.Password)) || !bytelen.Match([]byte(body.Password)) {
		return awsutils.RESTResponse(400, auth.CORSHeaders, "invalid password"), nil
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

	err = ddb.PutUser(ctx, body.Id, string(hash))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return awsutils.RESTResponse(201, auth.CORSHeaders, ""), nil
}

func main() {
	lambda.Start(handler)
}
