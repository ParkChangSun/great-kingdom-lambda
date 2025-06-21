package sqs

import (
	"context"
	"encoding/json"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/game"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var _sqsClient *sqs.Client

func sqsClient(ctx context.Context) *sqs.Client {
	if _sqsClient == nil {
		cfg, _ := config.LoadDefaultConfig(ctx)
		_sqsClient = sqs.NewFromConfig(cfg)
	}
	return _sqsClient
}

func SendToQueue(ctx context.Context, record Record, groupId string) error {
	body, _ := json.Marshal(record)

	_, err := sqsClient(ctx).SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(vars.GAME_TABLE_EVENT_QUEUE),
		MessageBody:    aws.String(string(body)),
		MessageGroupId: aws.String(groupId),
	})

	return err
}

type GameTableEvent struct {
	EventType vars.CLIENTEVENTTYPE
	game.Move
	Resign bool
}

type Record struct {
	GameTableEvent
	ddb.Connection
	Timestamp int64
}
