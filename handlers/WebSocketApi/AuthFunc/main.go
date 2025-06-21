package main

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"
	"great-kingdom-lambda/lib/sugarlogger"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	sugar := sugarlogger.GetSugar()
	defer sugar.Sync()

	connRepo := ddb.NewConnectionRepository()

	msg := struct{ Authorization string }{}
	json.Unmarshal([]byte(req.Body), &msg)

	conn, err := connRepo.Get(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	_, claims, err := auth.ParseToken(msg.Authorization)
	if err != nil {
		ws.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, vars.WebsocketPayload{EventType: vars.AUTHBROADCAST, Auth: false})
		ws.DeleteWebSocket(ctx, req.RequestContext.ConnectionID)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	if conn.UserId != claims.Subject {
		// this should not happen
		return events.APIGatewayProxyResponse{}, err
	}

	conn.Authorized = true
	err = connRepo.Put(ctx, conn)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	ws.SendWebsocketMessage(ctx, req.RequestContext.ConnectionID, vars.WebsocketPayload{EventType: vars.AUTHBROADCAST, Auth: true})

	if conn.GameTableId == "globalchat" {
		conns, err := ddb.NewConnectionRepository().Query(ctx)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		users := []string{}
		for _, v := range conns {
			users = append(users, v.UserId)
		}
		for _, v := range conns {
			ws.SendWebsocketMessage(ctx, v.Id, struct {
				EventType string
				Users     []string
			}{EventType: "USERS", Users: users})
		}
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r := sqs.Record{
		GameTableEvent: sqs.GameTableEvent{EventType: vars.TABLEJOINEVENT},
		Connection:     conn,
		Timestamp:      req.RequestContext.RequestTimeEpoch,
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
