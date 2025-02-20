package ddb

import (
	"context"
	"log"
	"os"
	"sam-app/awsutils"
	"sam-app/game"
	"sam-app/vars"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type GameLobbyDDBItem struct {
	GameLobbyId   string
	GameLobbyName string
	Connections   []ConnectionDDBItem
	Players       []string
	CoinToss      []string
	Game          game.Game
}

func GameLobbyDDBItemKey(gameLobbyId string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ GameLobbyId string }{gameLobbyId})
	return k
}

func PutLobbyInPool(ctx context.Context, lobbyName string) (string, error) {
	id := uuid.New().String()
	item, _ := attributevalue.MarshalMap(GameLobbyDDBItem{
		GameLobbyId:   id,
		GameLobbyName: lobbyName,
		Players:       []string{},
		Connections:   []ConnectionDDBItem{},
		CoinToss:      []string{},
	})

	_, err := client(ctx).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Item:      item,
	})

	return id, err
}

func (l GameLobbyDDBItem) DeleteFromPool(ctx context.Context) error {
	_, err := client(ctx).DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:       GameLobbyDDBItemKey(l.GameLobbyId),
	})

	return err
}

func GetGameLobby(ctx context.Context, gameLobbyId string) (GameLobbyDDBItem, error) {
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:       GameLobbyDDBItemKey(gameLobbyId),
	})
	if err != nil {
		return GameLobbyDDBItem{}, err
	}
	if query.Item == nil {
		return GameLobbyDDBItem{}, ErrItemNotFound
	}

	l := GameLobbyDDBItem{}
	attributevalue.UnmarshalMap(query.Item, &l)
	return l, nil
}

func ScanGameLobby(ctx context.Context) ([]GameLobbyDDBItem, error) {
	out, err := client(ctx).Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
	})
	if err != nil {
		return nil, err
	}

	items := []GameLobbyDDBItem{}
	attributevalue.UnmarshalListOfMaps(out.Items, &items)
	return items, nil
}

func (l GameLobbyDDBItem) SyncGame(ctx context.Context) error {
	update := expression.Set(expression.Name("Game"), expression.Value(l.Game))
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := client(ctx).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       GameLobbyDDBItemKey(l.GameLobbyId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (l GameLobbyDDBItem) SyncConnections(ctx context.Context) error {
	update := expression.Set(
		expression.Name("Connections"),
		expression.Value(l.Connections),
	).Set(
		expression.Name("Players"),
		expression.Value(l.Players),
	).Set(
		expression.Name("CoinToss"),
		expression.Value(l.CoinToss),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := client(ctx).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
		Key:                       GameLobbyDDBItemKey(l.GameLobbyId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (l GameLobbyDDBItem) ProcessGameResult(ctx context.Context, winner int) error {
	win, _ := GetUser(ctx, l.CoinToss[winner])
	lose, _ := GetUser(ctx, l.CoinToss[(winner+1)%2])

	item := RecentGame{BlueId: l.CoinToss[0], OrangeId: l.CoinToss[1], WinnerId: l.CoinToss[winner]}
	win.RecentGames = append(win.RecentGames, item)
	if len(win.RecentGames) > 10 {
		win.RecentGames = win.RecentGames[1:]
	}
	lose.RecentGames = append(win.RecentGames, item)
	if len(lose.RecentGames) > 10 {
		lose.RecentGames = lose.RecentGames[1:]
	}

	win.W++
	lose.L++

	win.SyncRecord(ctx)
	lose.SyncRecord(ctx)
	return nil
}

func (l GameLobbyDDBItem) StartNewGame() {
	l.Game.StartNewGame()
	l.CoinToss = l.Players
	if time.Now().Nanosecond()%2 == 0 {
		slices.Reverse(l.CoinToss)
	}
}

type LobbyBroadcastPayload struct {
	EventType         string
	*GameLobbyDDBItem `json:",omitempty"`
	Chat              string `json:",omitempty"`
}

func (s GameLobbyDDBItem) BroadcastChat(ctx context.Context, chat string) {
	s.Broadcast(ctx, LobbyBroadcastPayload{
		EventType: vars.CHATEVENT,
		Chat:      chat,
	})
}

func (s GameLobbyDDBItem) BroadcastGame(ctx context.Context) {
	s.Broadcast(ctx, LobbyBroadcastPayload{
		EventType:        vars.GAMEEVENT,
		GameLobbyDDBItem: &s,
	})
}

func (s GameLobbyDDBItem) Broadcast(ctx context.Context, payload LobbyBroadcastPayload) {
	for _, c := range s.Connections {
		err := awsutils.SendWebsocketMessage(ctx, c.ConnectionId, payload)
		if err != nil {
			log.Print(err)
		}
	}
}
