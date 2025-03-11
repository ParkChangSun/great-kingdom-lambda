package main

import (
	"context"
	"encoding/json"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := struct{ GameTableName string }{}
	err := json.Unmarshal([]byte(req.Body), &body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if body.GameTableName == "" {
		return awsutils.RESTResponse(400, auth.CORSHeaders, ""), nil
	}

	id, err := ddb.PutGameTable(ctx, body.GameTableName)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	b, _ := json.Marshal(struct{ GameTableId string }{GameTableId: id})
	return awsutils.RESTResponse(201, auth.CORSHeaders, string(b)), nil
}

func main() {
	lambda.Start(handler)
}
