package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"

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
		ws.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.AUTHBROADCAST, Auth: false})
		ws.DeleteWebSocket(ctx, req.RequestContext.ConnectionID)
		ddb.DeleteConnInPool(ctx, conn.ConnectionId)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	conn.UserId = claims.Subject
	err = conn.UpdateUserId(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	ws.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, ddb.GameTableBroadcastPayload{EventType: vars.AUTHBROADCAST, Auth: true})

	if conn.GameTableId == "globalchat" {
		ws.SendWebsocketMessage(ctx, conn.ConnectionId, ddb.GameTableBroadcastPayload{EventType: vars.CHATBROADCAST, Chat: "Connected."})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r := sqs.Record{
		GameTableEvent:    sqs.GameTableEvent{EventType: vars.TABLEJOINEVENT},
		ConnectionDDBItem: conn,
		Timestamp:         req.RequestContext.RequestTimeEpoch,
	}
	err = sqs.SendToQueue(ctx, r, r.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
