package ddb

import (
	"context"
	"great-kingdom-lambda/lib/game"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Player struct {
	Id            string
	RemainingTime int64
}

type GameSession struct {
	Id          string
	Name        string
	Connections []Connection
	Players     []*Player
	// even=>b o, odd=>o b
	CoinToss     int64
	LastMoveTime int64
	GameTable    *game.GameTable `json:",omitempty"`
}

func (s *GameSession) StartNewGame() {
	s.GameTable = game.NewGameTable()

	now := time.Now().UnixMilli()
	s.CoinToss = now
	s.LastMoveTime = now

	t, _ := time.ParseDuration("5m")
	s.Players[0].RemainingTime = t.Milliseconds()
	s.Players[1].RemainingTime = t.Milliseconds()
}

func (s GameSession) Playing() bool {
	return s.GameTable != nil && s.GameTable.Result == ""
}

func (s GameSession) CurrentTurnPlayer() *Player {
	return s.Players[(s.GameTable.Turn+int(s.CoinToss)+1)%2]
}

type GameSessionBroadcastPayload struct {
	Chat        string
	GameSession *GameSession
}

func (s *GameSession) Broadcast(ctx context.Context, chat string) {
	for _, c := range s.Connections {
		err := ws.SendWebsocketMessage(ctx, c.Id, GameSessionBroadcastPayload{Chat: chat, GameSession: s})
		if err != nil {
			log.Print(err)
		}
	}
}

type SessionRepository struct {
	client *dynamodb.Client
}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{client: _client}
}

func GameSessionKey(id string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ Id string }{id})
	return k
}

func (r *SessionRepository) Get(ctx context.Context, id string) (GameSession, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(vars.GAME_SESSION_DYNAMODB),
		Key:       GameSessionKey(id),
	})
	if err != nil {
		return GameSession{}, err
	}

	if out.Item == nil {
		return GameSession{}, &ItemNotFoundError{vars.GAME_SESSION_DYNAMODB, id}
	}

	l := GameSession{}
	attributevalue.UnmarshalMap(out.Item, &l)
	return l, nil
}

func (r *SessionRepository) Put(ctx context.Context, session GameSession) error {
	item, _ := attributevalue.MarshalMap(session)

	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.GAME_SESSION_DYNAMODB),
		Item:      item,
	})

	return err
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(vars.GAME_SESSION_DYNAMODB),
		Key:       GameSessionKey(id),
	})

	return err
}

func (r *SessionRepository) Scan(ctx context.Context) ([]GameSession, error) {
	out, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(vars.GAME_SESSION_DYNAMODB),
	})
	if err != nil {
		return nil, err
	}

	items := []GameSession{}
	attributevalue.UnmarshalListOfMaps(out.Items, &items)
	return items, nil
}

func (s GameSession) ProcessGameResult(ctx context.Context, winPlayerIndex int) error {
	userRepo := NewUserRepository()
	win, _ := userRepo.Get(ctx, s.Players[winPlayerIndex].Id)
	lose, _ := userRepo.Get(ctx, s.Players[(winPlayerIndex+1)%2].Id)

	item := RecentGame{
		BlueId:      s.Players[s.CoinToss%2].Id,
		OrangeId:    s.Players[(s.CoinToss+1)%2].Id,
		Result:      s.GameTable.Result,
		CreatedDate: time.Now().Format(time.RFC3339),
	}
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

	userRepo.Put(ctx, win)
	userRepo.Put(ctx, lose)
	return nil
}
