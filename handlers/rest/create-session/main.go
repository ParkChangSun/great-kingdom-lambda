package main

import (
	"context"
	"encoding/json"
	"sam-app/auth"
	"sam-app/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := struct {
		GameSessionName string
	}{}
	err := json.Unmarshal([]byte(req.Body), &body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if body.GameSessionName == "" {
		return events.APIGatewayProxyResponse{}, err
	}

	id, err := ddb.PutLobbyInPool(ctx, body.GameSessionName)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	b, _ := json.Marshal(struct{ GameSessionId string }{GameSessionId: id})
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    auth.CORSHeaders,
		Body:       string(b),
	}, nil
}

func main() {
	lambda.Start(handler)
}
