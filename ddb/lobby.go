package ddb

import (
	"context"
	"log"
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

type GameTableDDBItem struct {
	GameTableId   string
	GameTableName string
	Connections   []ConnectionDDBItem
	Players       []string
	CoinToss      []string
	Game          game.Game
}

func GameTableDDBKey(gameTableId string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ GameTableId string }{gameTableId})
	return k
}

func PutGameTable(ctx context.Context, tableName string) (string, error) {
	id := uuid.New().String()
	item, _ := attributevalue.MarshalMap(GameTableDDBItem{
		GameTableId:   id,
		GameTableName: tableName,
		Players:       []string{},
		Connections:   []ConnectionDDBItem{},
		CoinToss:      []string{},
	})

	_, err := client(ctx).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.GAME_TABLE_DYNAMODB),
		Item:      item,
	})

	return id, err
}

func DeleteGameTable(ctx context.Context, tableId string) error {
	_, err := client(ctx).DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(vars.GAME_TABLE_DYNAMODB),
		Key:       GameTableDDBKey(tableId),
	})

	return err
}

func GetGameTable(ctx context.Context, gameTableId string) (GameTableDDBItem, error) {
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(vars.GAME_TABLE_DYNAMODB),
		Key:       GameTableDDBKey(gameTableId),
	})
	if err != nil {
		return GameTableDDBItem{}, err
	}
	if query.Item == nil {
		return GameTableDDBItem{}, ErrItemNotFound
	}

	l := GameTableDDBItem{}
	attributevalue.UnmarshalMap(query.Item, &l)
	return l, nil
}

func ScanGameTable(ctx context.Context) ([]GameTableDDBItem, error) {
	out, err := client(ctx).Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(vars.GAME_TABLE_DYNAMODB),
	})
	if err != nil {
		return nil, err
	}

	items := []GameTableDDBItem{}
	attributevalue.UnmarshalListOfMaps(out.Items, &items)
	return items, nil
}

func (l GameTableDDBItem) SyncGame(ctx context.Context) error {
	update := expression.Set(expression.Name("Game"), expression.Value(l.Game))
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := client(ctx).UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(vars.GAME_TABLE_DYNAMODB),
		Key:                       GameTableDDBKey(l.GameTableId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (l GameTableDDBItem) SyncConnections(ctx context.Context) error {
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
		TableName:                 aws.String(vars.GAME_TABLE_DYNAMODB),
		Key:                       GameTableDDBKey(l.GameTableId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	return err
}

func (l GameTableDDBItem) ProcessGameResult(ctx context.Context, winner int) error {
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

func (l GameTableDDBItem) StartNewGame() {
	l.Game.StartNewGame()
	l.CoinToss = l.Players
	if time.Now().Nanosecond()%2 == 0 {
		slices.Reverse(l.CoinToss)
	}
}

type GameTableBroadcastPayload struct {
	EventType         string
	*GameTableDDBItem `json:",omitempty"`
	Chat              string `json:",omitempty"`
}

func (s GameTableDDBItem) BroadcastChat(ctx context.Context, chat string) {
	s.Broadcast(ctx, GameTableBroadcastPayload{
		EventType: vars.CHATEVENT,
		Chat:      chat,
	})
}

func (s GameTableDDBItem) BroadcastGame(ctx context.Context) {
	s.Broadcast(ctx, GameTableBroadcastPayload{
		EventType:        vars.GAMEEVENT,
		GameTableDDBItem: &s,
	})
}

func (s GameTableDDBItem) Broadcast(ctx context.Context, payload GameTableBroadcastPayload) {
	for _, c := range s.Connections {
		err := awsutils.SendWebsocketMessage(ctx, c.ConnectionId, payload)
		if err != nil {
			log.Print(err)
		}
	}
}
