package main

import (
	"context"
	"encoding/json"
	"fmt"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"
	"log"
	"time"

	"slices"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func joinEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	if slices.ContainsFunc(l.Connections, func(c ddb.ConnectionDDBItem) bool { return c.UserId == record.UserId }) {
		ddb.DeleteConnInPool(ctx, record.ConnectionId)
		return ws.DeleteWebSocket(ctx, record.ConnectionId)
	}

	l.Connections = append(l.Connections, record.ConnectionDDBItem)
	if len(l.Players) < 2 {
		l.Players = append(l.Players, record.UserId)
	}

	err := l.SyncConnections(ctx)
	if err != nil {
		return err
	}

	l.BroadcastTable(ctx)
	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " has joined the game."))

	return nil
}

func leaveEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	l.Connections = slices.DeleteFunc(l.Connections, func(u ddb.ConnectionDDBItem) bool { return u.UserId == record.UserId })
	if len(l.Connections) == 0 {
		return ddb.DeleteGameTable(ctx, l.GameTableId)
	}

	l.Players = slices.DeleteFunc(l.Players, func(u string) bool { return u == record.UserId })
	if l.Game.Playing && len(l.Players) < 2 {
		l.ProcessGameResult(ctx, slices.IndexFunc(l.Game.CoinToss, func(u string) bool { return u == l.Players[0] }))
		l.BroadcastChat(ctx, fmt.Sprint(l.Players[0], " surrender."))

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

	l.BroadcastTable(ctx)
	l.BroadcastChat(ctx, fmt.Sprint(record.UserId, " has left the game."))

	return nil
}

func slotEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	if l.Game.Playing {
		return nil
	}

	if slices.ContainsFunc(l.Players, func(u string) bool { return u == record.UserId }) {
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

	l.BroadcastTable(ctx)

	return nil
}

func startEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	if record.UserId != l.Players[0] || l.Game.Playing || len(l.Players) != 2 {
		return nil
	}

	l.Game.StartNewGame(l.Players)

	err := l.SyncGame(ctx)
	if err != nil {
		return err
	}

	l.BroadcastTable(ctx)
	l.BroadcastChat(ctx, fmt.Sprint("Game start. ", l.Game.CoinToss[0], " plays first."))

	return nil
}

func gameEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	if !l.Game.Playing || l.Game.CoinToss[(l.Game.Turn-1)%2] != record.UserId {
		return nil
	}

	now := time.Now().UnixMilli()
	l.Game.RemainingTime[(l.Game.Turn-1)%2] -= now - l.Game.LastMoveTime
	l.Game.LastMoveTime = now

	if l.Game.RemainingTime[(l.Game.Turn-1)%2] <= 0 || record.Surrender {
		l.Game.Playing = false
		err := l.SyncGame(ctx)
		if err != nil {
			return err
		}
		l.ProcessGameResult(ctx, l.Game.Turn%2)
		l.BroadcastTable(ctx)
		l.BroadcastChat(ctx, fmt.Sprint("Game over. ", l.Game.CoinToss[(l.Game.Turn+1)%2], " surrender"))
		return nil
	}

	if record.Pass {
		l.Game.Pass()
	} else if l.Game.CellPlayable(record.Move) {
		l.Game.Move(record.Move)
	} else {
		return nil
	}

	if !l.Game.Playing {
		b, o, sieged, w := l.Game.CountTerritory()
		l.ProcessGameResult(ctx, w)
		if sieged {
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", l.Game.CoinToss[w], " sieged its opponent."))
		} else {
			l.BroadcastChat(ctx, fmt.Sprint("Game over. ", b, " : ", o, "(+2.5)", " ", l.Game.CoinToss[w], " won."))
		}
	}

	err := l.SyncGame(ctx)
	if err != nil {
		return err
	}

	l.BroadcastTable(ctx)

	return nil
}

func kickEvent(ctx context.Context, record sqs.Record, l ddb.GameTableDDBItem) error {
	if !l.Game.Playing || l.Game.CoinToss[l.Game.Turn%2] != record.UserId || l.Game.RemainingTime[(l.Game.Turn+1)%2] > time.Now().UnixMilli()-l.Game.LastMoveTime {
		return nil
	}

	l.Game.Playing = false
	l.ProcessGameResult(ctx, l.Game.Turn%2)
	err := l.SyncGame(ctx)
	if err != nil {
		return err
	}

	kickedId := l.Game.CoinToss[(l.Game.Turn+1)%2]
	l.Players = slices.DeleteFunc(l.Players, func(u string) bool { return u == kickedId })
	i := slices.IndexFunc(l.Connections, func(u ddb.ConnectionDDBItem) bool { return u.UserId == kickedId })
	err = ws.DeleteWebSocket(ctx, l.Connections[i].ConnectionId)
	if err != nil {
		return err
	}
	l.Connections = slices.Delete(l.Connections, i, i+1)

	l.SyncConnections(ctx)

	l.BroadcastChat(ctx, fmt.Sprint(l.Game.CoinToss[(l.Game.Turn+1)%2], " timeout"))
	l.BroadcastTable(ctx)

	return nil
}

func handler(ctx context.Context, req events.SQSEvent) {
	for _, record := range req.Records {
		r := sqs.Record{}
		json.Unmarshal([]byte(record.Body), &r)
		if r.UserId == "" {
			continue
		}

		l, err := ddb.GetGameTable(ctx, r.GameTableId)
		if err != nil {
			ddb.DeleteConnInPool(ctx, r.ConnectionId)
			err = ws.DeleteWebSocket(ctx, r.ConnectionId)
			if err != nil {
				log.Print(err)
			}
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

		case vars.TABLESTARTEVENT:
			err = startEvent(ctx, r, l)

		case vars.KICK:
			kickEvent(ctx, r, l)
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
