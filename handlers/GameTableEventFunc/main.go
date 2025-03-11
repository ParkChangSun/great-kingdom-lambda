package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/vars"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func joinEvent(ctx context.Context, record ddb.Record, l ddb.GameTableDDBItem) error {
	if slices.ContainsFunc(l.Connections, func(c ddb.ConnectionDDBItem) bool { return c.UserId == record.UserId }) {
		log.Print("dup conn detected")
		awsutils.DeleteWebSocket(ctx, record.ConnectionId)
		return nil
	}

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

func leaveEvent(ctx context.Context, record ddb.Record, l ddb.GameTableDDBItem) error {
	l.Connections = slices.DeleteFunc(l.Connections, func(u ddb.ConnectionDDBItem) bool { return u.UserId == record.UserId })
	if len(l.Connections) == 0 {
		return ddb.DeleteGameTable(ctx, l.GameTableId)
	}

	l.Players = slices.DeleteFunc(l.Players, func(u string) bool { return u == record.UserId })
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

func chatEvent(ctx context.Context, record ddb.Record, l ddb.GameTableDDBItem) error {
	msg := strings.Trim(record.Chat, " ")
	if msg == "" {
		return nil
	}

	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " : ", msg))

	return nil
}

func slotEvent(ctx context.Context, record ddb.Record, l ddb.GameTableDDBItem) error {
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

func gameEvent(ctx context.Context, record ddb.Record, l ddb.GameTableDDBItem) error {
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

	if record.Move.Pass {
		l.Game.Pass()
	} else if l.Game.CellPlayable(record.Move.Point) {
		l.Game.Move(record.Move.Point)
	} else {
		return nil
	}

	if !l.Game.Playing {
		b, o, sieged, w := l.Game.CountTerritory()
		l.ProcessGameResult(ctx, w)
		if sieged {
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", l.CoinToss[w], " sieged its opponent."))
		} else {
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", b, " : ", o, "(+3)", " ", l.CoinToss[w], " won."))
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
		r := ddb.Record{}
		json.Unmarshal([]byte(record.Body), &r)
		if r.UserId == "" {
			continue
		}

		l, err := ddb.GetGameTable(ctx, r.GameTableId)
		if err != nil {
			log.Print(err)
			continue
		}

		switch r.EventType {
		case vars.TABLEJOINEVENT:
			err = joinEvent(ctx, r, l)

		case vars.TABLELEAVEEVENT:
			err = leaveEvent(ctx, r, l)

		case vars.TABLEMOVEEVENT:
			err = gameEvent(ctx, r, l)

		case vars.TABLESLOTEVENT:
			err = slotEvent(ctx, r, l)

		case vars.TABLECHATEVENT:
			err = chatEvent(ctx, r, l)
		}

		if err != nil {
			log.Print(err)
			l.BroadcastChat(ctx, err.Error())
			continue
		}
	}
}

func main() {
	lambda.Start(handler)
}
