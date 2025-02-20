package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sam-app/ddb"
	"sam-app/vars"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func joinEvent(ctx context.Context, record ddb.Record, l ddb.GameLobbyDDBItem) error {
	l.Connections = append(l.Connections, record.ConnectionDDBItem)
	if len(l.Players) < 2 {
		l.Players = append(l.Players, record.UserId)
	}

	err := l.SyncConnections(ctx)
	if err != nil {
		return err
	}

	l.BroadcastGame(ctx)
	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " has joined the game."))

	return nil
}

func leaveEvent(ctx context.Context, record ddb.Record, l ddb.GameLobbyDDBItem) error {
	l.Players = slices.DeleteFunc(l.Players, func(u string) bool { return u == record.UserId })
	l.Connections = slices.DeleteFunc(l.Connections, func(u ddb.ConnectionDDBItem) bool { return u.UserId == record.UserId })
	if len(l.Connections) == 0 {
		return l.DeleteFromPool(ctx)
	}

	if l.Game.Playing && len(l.Players) < 2 {
		l.ProcessGameResult(ctx, slices.IndexFunc(l.CoinToss, func(u string) bool { return u == l.Players[0] }))
		l.BroadcastChat(ctx, fmt.Sprint(l.Players[0], " won."))

		l.Game.Playing = false
		err := l.SyncGame(ctx)
		if err != nil {
			return err
		}
	}

	err := l.SyncConnections(ctx)
	if err != nil {
		return err
	}

	l.BroadcastGame(ctx)
	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " has left the game."))

	return nil
}

func chatEvent(ctx context.Context, record ddb.Record, l ddb.GameLobbyDDBItem) error {
	msg := strings.Trim(record.Chat, " ")
	if msg == "" {
		return nil
	}

	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " : ", msg))

	return nil
}

func slotEvent(ctx context.Context, record ddb.Record, l ddb.GameLobbyDDBItem) error {
	if l.Game.Playing {
		return nil
	} else if slices.ContainsFunc(l.Players, func(u string) bool { return u == record.UserId }) {
		l.Players = slices.DeleteFunc(l.Players, func(u string) bool { return u == record.UserId })
	} else if len(l.Players) < 2 {
		l.Players = append(l.Players, record.UserId)
	} else {
		return nil
	}

	err := l.SyncConnections(ctx)
	if err != nil {
		return err
	}

	l.BroadcastGame(ctx)

	return nil
}

func gameEvent(ctx context.Context, record ddb.Record, l ddb.GameLobbyDDBItem) error {
	if record.Move.Start && record.UserId == l.Players[0] && !l.Game.Playing && len(l.Players) == 2 {
		l.StartNewGame()
		err := l.SyncGame(ctx)
		if err != nil {
			return err
		}
		err = l.SyncConnections(ctx)
		if err != nil {
			return err
		}

		l.BroadcastGame(ctx)
		l.BroadcastChat(ctx, fmt.Sprint("Game start. ", l.CoinToss[0], " plays first."))

		return nil
	}

	if !l.Game.Playing || l.CoinToss[(l.Game.Turn-1)%2] != record.UserId {
		return nil
	}

	if record.Move.Pass && l.Game.Pass() {
		b, o, w := l.Game.CountTerritory()
		l.ProcessGameResult(ctx, w)
		l.BroadcastChat(ctx, fmt.Sprint("Game over. ", b, " : ", o, "(+3)", " ", l.CoinToss[0], " won."))
	} else {
		finished, sieged, err := l.Game.Move(record.Move.Point)
		if err != nil {
			return err
		}
		if sieged {
			l.ProcessGameResult(ctx, (l.Game.Turn-1)%2)
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", l.CoinToss[(l.Game.Turn-1)%2], " won."))
		} else if finished {
			b, o, w := l.Game.CountTerritory()
			l.ProcessGameResult(ctx, w)
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", b, " : ", o, "(+3)", " ", l.CoinToss[0], " won."))
		}
	}

	err := l.SyncGame(ctx)
	if err != nil {
		return err
	}

	l.BroadcastGame(ctx)

	return nil
}

func handler(ctx context.Context, req events.SQSEvent) {
	for _, record := range req.Records {
		var err error

		r := ddb.Record{}
		json.Unmarshal([]byte(record.Body), &r)

		l, err := ddb.GetGameLobby(ctx, r.GameSessionId)
		if err != nil {
			log.Print(err)
			continue
		}

		switch r.EventType {
		case vars.JOINEVENT:
			err = joinEvent(ctx, r, l)

		case vars.LEAVEEVENT:
			err = leaveEvent(ctx, r, l)

		case vars.GAMEEVENT:
			err = gameEvent(ctx, r, l)

		case vars.SLOTEVENT:
			err = slotEvent(ctx, r, l)

		case vars.CHATEVENT:
			err = chatEvent(ctx, r, l)
		}

		if err != nil {
			log.Print(err)
			continue
		}
	}
}

func main() {
	lambda.Start(handler)
}
