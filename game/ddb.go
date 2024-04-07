package game

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func GetGameSession(ctx context.Context, gameSessionId string) (GameSession, error) {
	gameSession := GameSession{}
	cfg, _ := config.LoadDefaultConfig(ctx)

	out, err := dynamodb.NewFromConfig(cfg).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:       GetGameSessionDynamoDBKey(gameSessionId),
	})
	if err != nil {
		return gameSession, err
	}

	attributevalue.UnmarshalMap(out.Item, &gameSession)
	return gameSession, nil
}
