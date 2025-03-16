package main

import (
	"context"
	"encoding/json"
	"sam-app/auth"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/vars"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	msg := struct{ Authorization string }{}
	json.Unmarshal([]byte(req.Body), &msg)

	conn, err := ddb.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	_, claims, err := auth.ParseToken(msg.Authorization)
	if err != nil {
		awsutils.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.AUTHBROADCAST, Auth: false})
		awsutils.DeleteWebSocket(ctx, req.RequestContext.ConnectionID)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	conn.UserId = claims.Subject
	err = conn.UpdateUserId(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	awsutils.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.AUTHBROADCAST, Auth: true})

	if conn.GameTableId == "globalchat" {
		awsutils.SendWebsocketMessage(ctx, conn.ConnectionId, ddb.GameTableBroadcastPayload{EventType: vars.CHATBROADCAST, Chat: "Connected."})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r := ddb.Record{
		EventType:         vars.TABLEJOINEVENT,
		ConnectionDDBItem: conn,
		Timestamp:         req.RequestContext.RequestTimeEpoch,
	}
	err = awsutils.SendToQueue(ctx, r, r.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
