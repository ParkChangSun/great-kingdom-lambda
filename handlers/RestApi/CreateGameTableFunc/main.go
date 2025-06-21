package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := struct{ GameTableName string }{}
	json.Unmarshal([]byte(req.Body), &body)

	if body.GameTableName == "" {
		return auth.RESTResponse(400, auth.CORSHeaders, ""), nil
	}

	id := uuid.New().String()
	err := ddb.NewSessionRepository().Put(ctx, ddb.GameSession{
		Id:          id,
		Name:        body.GameTableName,
		Players:     []*ddb.Player{},
		Connections: []ddb.Connection{},
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	b, _ := json.Marshal(struct{ GameTableId string }{GameTableId: id})
	return auth.RESTResponse(201, auth.CORSHeaders, string(b)), nil
}

func main() {
	lambda.Start(handler)
}
