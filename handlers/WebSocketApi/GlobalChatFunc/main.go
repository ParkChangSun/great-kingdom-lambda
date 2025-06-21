package main

import (
	"bytes"
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"
	"log"
	"net/http"

	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	data := struct{ Chat string }{}
	json.Unmarshal([]byte(req.Body), &data)

	sender, err := ddb.NewConnectionRepository().Get(ctx, req.RequestContext.ConnectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	receivers, err := ddb.NewConnectionRepository().Query(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	for _, v := range receivers {
		ws.SendWebsocketMessage(ctx, v.Id, vars.WebsocketPayload{
			EventType: vars.CHATBROADCAST,
			Chat:      strings.Join([]string{sender.UserId, ":", data.Chat}, " "),
		})
	}

	d, _ := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: strings.Join([]string{sender.UserId, ":", data.Chat}, " ")})
	r, err := http.Post(vars.DISCORD_WEBHOOK, "application/json", bytes.NewBuffer(d))
	if err != nil {
		log.Print(err)
	}
	defer r.Body.Close()

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
