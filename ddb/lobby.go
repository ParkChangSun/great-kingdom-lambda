package ddb

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sam-app/awsutils"
	"sam-app/game"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type User struct {
	ConnectionId string
	UserId       string
}

type GameSessionDDBItem struct {
	GameSessionId      string
	GameSessionName    string
	CurrentConnections []User
	Players            []User
	Game               game.Game
}

func (s GameSessionDDBItem) UpdateGame(ctx context.Context) error {
	update := expression.Set(expression.Name("Game"), expression.Value(s.Game))
	condition := expression.AttributeExists(expression.Name("GameSessionId"))
	expr, _ := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{s.GameSessionId})

	_, err := client(ctx).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       key,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (s GameSessionDDBItem) UpdatePlayers(ctx context.Context) error {
	update := expression.Set(
		expression.Name("CurrentConnections"),
		expression.Value(s.CurrentConnections),
	).Set(
		expression.Name("Players"),
		expression.Value(s.Players),
	)
	condition := expression.AttributeExists(expression.Name("GameSessionId"))
	expr, _ := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{s.GameSessionId})

	_, err := client(ctx).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       key,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (s *GameSessionDDBItem) StartNewGame(blueId string, orangeId string) {
	s.Game.StartNewGame(blueId, orangeId)
}

func GetGameSession(ctx context.Context, gameSessionId string) (GameSessionDDBItem, error) {
	key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{gameSessionId})

	out, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:       key,
	})
	if err != nil {
		return GameSessionDDBItem{}, err
	}

	gameSession := GameSessionDDBItem{}
	attributevalue.UnmarshalMap(out.Item, &gameSession)
	return gameSession, nil
}

func (s GameSessionDDBItem) BroadCastWebSocketMessage(ctx context.Context, payload any) {
	data, _ := json.Marshal(payload)

	cfg, _ := config.LoadDefaultConfig(ctx)
	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
	})

	for _, c := range s.CurrentConnections {
		_, err := wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(c.ConnectionId),
			Data:         data,
		})
		if err != nil {
			log.Print(err)
		}
	}
}

func (s GameSessionDDBItem) BroadCastChat(ctx context.Context, chat string) {
	s.BroadCastWebSocketMessage(ctx, struct {
		EventType string
		Chat      string
	}{
		EventType: awsutils.CHATEVENT,
		Chat:      chat,
	})
}

func (s GameSessionDDBItem) BroadCastGame(ctx context.Context) {
	s.BroadCastWebSocketMessage(ctx, struct {
		EventType string
		Game      game.Game
	}{
		EventType: awsutils.GAMEEVENT,
		Game:      s.Game,
	})
}

func (s GameSessionDDBItem) BroadCastUser(ctx context.Context) {
	s.BroadCastWebSocketMessage(ctx, struct {
		EventType          string
		Players            []User
		CurrentConnections []User
		GameSessionName    string
		GameSessionId      string
	}{
		EventType:          awsutils.USEREVENT,
		Players:            s.Players,
		CurrentConnections: s.CurrentConnections,
		GameSessionName:    s.GameSessionName,
		GameSessionId:      s.GameSessionId,
	})
}

func (s GameSessionDDBItem) UpdateGameResult(ctx context.Context, winner int) error {
	blue, err := GetUser(ctx, s.Game.PlayersId[0])
	if err != nil {
		return err
	}
	orange, err := GetUser(ctx, s.Game.PlayersId[1])
	if err != nil {
		return err
	}

	if winner == 0 {
		blue.W++
		orange.L++
	} else if winner == 1 {
		blue.L++
		orange.W++
	}

	err = blue.SyncRecord(ctx)
	if err != nil {
		return err
	}
	err = orange.SyncRecord(ctx)
	if err != nil {
		return err
	}

	return nil
}
