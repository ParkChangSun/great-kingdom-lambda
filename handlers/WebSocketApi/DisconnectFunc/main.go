package main

import (
	"context"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"
	"great-kingdom-lambda/lib/sugarlogger"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	conn, err := ddb.NewConnectionRepository().Delete(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	sugarlogger.GetSugar().Info("disconnect ", conn.UserId, conn.Id)

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
		GameTableEvent: sqs.GameTableEvent{EventType: vars.TABLELEAVEEVENT},
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
