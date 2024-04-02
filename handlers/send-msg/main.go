package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sam-app/chat"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

func handler(ctx context.Context, req events.SQSEvent) error {
	log.Print(req.Records)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("API_ENDPOINT"))
	})

	payload := chat.Msg{}
	err = json.Unmarshal([]byte(req.Records[0].Body), &payload)
	if err != nil {
		return err
	}

	_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(payload.Id),
		Data:         []byte(payload.Chat),
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
