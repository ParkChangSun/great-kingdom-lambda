package main

import (
	"context"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if _, ok := req.QueryStringParameters["GameTableId"]; !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    auth.CORSHeaders,
			Body:       "id not found",
		}, nil
	}

	if req.QueryStringParameters["GameTableId"] == "globalchat" {
		r := ddb.Record{
			EventType: vars.GLOBALCHAT,
			ConnectionDDBItem: ddb.ConnectionDDBItem{
				ConnectionId: req.RequestContext.ConnectionID,
				GameTableId:  req.QueryStringParameters["GameTableId"],
				UserId:       req.RequestContext.Authorizer.(map[string]any)["UserId"].(string),
			},
			Timestamp: req.RequestContext.RequestTimeEpoch,
		}
		err := ddb.PutConnInPool(ctx, r.ConnectionDDBItem)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r := ddb.Record{
		EventType: vars.JOINEVENT,
		ConnectionDDBItem: ddb.ConnectionDDBItem{
			ConnectionId: req.RequestContext.ConnectionID,
			GameTableId:  req.QueryStringParameters["GameTableId"],
			UserId:       req.RequestContext.Authorizer.(map[string]any)["UserId"].(string),
		},
		Timestamp: req.RequestContext.RequestTimeEpoch,
	}

	err := awsutils.SendToQueue(ctx, r, r.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
