package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sugarlogger"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	startKey := req.QueryStringParameters["StartKey"]
	userId := req.QueryStringParameters["UserId"]

	q, err := ddb.NewRecordRepository().Query(ctx, userId, startKey)
	if err != nil {
		sugarlogger.GetSugar().Error("getrecord lastkey and userid", startKey, userId, err)
		return events.APIGatewayProxyResponse{}, err
	}

	sugarlogger.GetSugar().Info("getrecord lastkey and userid", startKey, userId)

	body, _ := json.Marshal(q)
	return auth.RESTResponse(200, auth.CORSHeaders, string(body)), nil
}

func main() {
	lambda.Start(handler)
}
