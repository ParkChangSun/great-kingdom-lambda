package main

import (
	"context"
	"encoding/json"
	"fmt"
	"great-kingdom-lambda/lib/ddb"

	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	chat := struct{ Chat string }{}
	json.Unmarshal([]byte(req.Body), &chat)
	msg := strings.Trim(chat.Chat, " ")
	if msg == "" {
		return events.APIGatewayProxyResponse{}, nil
	}

	conn, err := ddb.GetConnection(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	l, err := ddb.GetGameTable(ctx, conn.GameTableId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	l.BroadcastChat(ctx, fmt.Sprint(conn.UserId, " : ", msg))

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
