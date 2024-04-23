package game

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	JOINEVENT  = "JOIN"
	LEAVEEVENT = "LEAVE"
	CHATEVENT  = "CHAT"
	GAMEEVENT  = "GAME"
	USEREVENT  = "USER"
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
	Game               Game
}

func GetGameSessionDynamoDBKey(gameSessionId string) map[string]types.AttributeValue {
	key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{gameSessionId})
	return key
}

func (s GameSessionDDBItem) UpdateGame(ctx context.Context) error {
	cfg, _ := config.LoadDefaultConfig(ctx)

	update := expression.Set(expression.Name("Game"), expression.Value(s.Game))
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := dynamodb.NewFromConfig(cfg).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       GetGameSessionDynamoDBKey(s.GameSessionId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s GameSessionDDBItem) UpdatePlayers(ctx context.Context) error {
	cfg, _ := config.LoadDefaultConfig(ctx)

	update := expression.Set(
		expression.Name("CurrentConnections"),
		expression.Value(s.CurrentConnections),
	).Set(
		expression.Name("Players"),
		expression.Value(s.Players),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := dynamodb.NewFromConfig(cfg).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       GetGameSessionDynamoDBKey(s.GameSessionId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GameSessionDDBItem) StartNewGame(blueId string, orangeId string) {
	s.Game = Game{
		Turn:     1,
		PassFlag: false,
		Playing:  true,
	}
	if time.Now().UnixMilli()%2 == 0 {
		s.Game.PlayersId = [2]string{blueId, orangeId}
	} else {
		s.Game.PlayersId = [2]string{orangeId, blueId}
	}
	s.Game.Board[4][4] = Neutral
}

func GetGameSession(ctx context.Context, gameSessionId string) (GameSessionDDBItem, error) {
	gameSession := GameSessionDDBItem{}
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

func (s GameSessionDDBItem) BroadCastWebSocketMessage(ctx context.Context, payload any) {
	data, _ := json.Marshal(payload)

	cfg, _ := config.LoadDefaultConfig(ctx)
	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("API_ENDPOINT"))
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
		EventType: CHATEVENT,
		Chat:      chat,
	})
}

func (s GameSessionDDBItem) BroadCastGame(ctx context.Context) {
	s.BroadCastWebSocketMessage(ctx, struct {
		EventType string
		Game      Game
	}{
		EventType: GAMEEVENT,
		Game:      s.Game,
	})
}

func (s GameSessionDDBItem) BroadCastUser(ctx context.Context) {
	s.BroadCastWebSocketMessage(ctx, struct {
		EventType          string
		Players            []User
		CurrentConnections []User
	}{
		EventType:          USEREVENT,
		Players:            s.Players,
		CurrentConnections: s.CurrentConnections,
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

	if winner == -1 {
		blue.D++
		orange.D++
	} else if winner == 0 {
		blue.W++
		orange.L++
	} else if winner == 1 {
		blue.L++
		orange.W++
	}

	err = blue.UpdateRecord(ctx)
	if err != nil {
		return err
	}
	err = orange.UpdateRecord(ctx)
	if err != nil {
		return err
	}

	return nil
}

type ConnectionDDBItem struct {
	ConnectionId  string
	Timestamp     int64
	GameSessionId string
	UserId        string
}

func GetConnectionDynamoDBKey(connectionId string) map[string]types.AttributeValue {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{connectionId})
	return key
}

func GetConnection(ctx context.Context, connectionId string) (ConnectionDDBItem, error) {
	record := ConnectionDDBItem{}
	cfg, _ := config.LoadDefaultConfig(ctx)

	query, err := dynamodb.NewFromConfig(cfg).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Key:       GetConnectionDynamoDBKey(connectionId),
	})
	if err != nil {
		return record, err
	}

	attributevalue.UnmarshalMap(query.Item, &record)
	return record, nil
}

type UserDDBItem struct {
	UserUUID     string
	UserId       string
	PasswordHash string
	W, L, D      int
}

func GetUser(ctx context.Context, userId string) (UserDDBItem, error) {
	item := UserDDBItem{}
	cfg, _ := config.LoadDefaultConfig(ctx)

	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: userId})
	query, err := dynamodb.NewFromConfig(cfg).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       k,
	})
	if err != nil {
		return item, err
	}

	attributevalue.UnmarshalMap(query.Item, &item)
	return item, nil
}

func (u UserDDBItem) UpdateRecord(ctx context.Context) error {
	cfg, _ := config.LoadDefaultConfig(ctx)

	update := expression.Set(
		expression.Name("W"),
		expression.Value(u.W),
	).Set(
		expression.Name("L"),
		expression.Value(u.L),
	).Set(
		expression.Name("D"),
		expression.Value(u.D),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: u.UserId})

	_, err := dynamodb.NewFromConfig(cfg).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("USER_DYNAMODB")),
		Key:                       k,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		return err
	}

	return nil
}

type GameMoveSQSRecord struct {
	Timestamp     int64
	ConnectionId  string
	UserId        string
	GameSessionId string
	Move          Move
}

type GameChatSQSRecord struct {
	Timestamp     int64
	ConnectionId  string
	UserId        string
	GameSessionId string
	Chat          string
}
