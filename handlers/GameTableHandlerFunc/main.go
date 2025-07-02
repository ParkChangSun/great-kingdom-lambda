package main

import (
	"context"
	"encoding/json"
	"fmt"
	"great-kingdom-lambda/lib/ddb"
	"great-kingdom-lambda/lib/sqs"
	"great-kingdom-lambda/lib/sugarlogger"
	"great-kingdom-lambda/lib/vars"
	"great-kingdom-lambda/lib/ws"
	"time"

	"slices"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func joinEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	sessionRepo := ddb.NewSessionRepository()

	if slices.ContainsFunc(s.Connections, func(c ddb.Connection) bool { return c.UserId == record.UserId }) {
		sessionRepo.Delete(ctx, record.Id)
		return ws.DeleteWebSocket(ctx, record.Id)
	}

	s.Connections = append(s.Connections, record.Connection)
	if len(s.Players) < 2 {
		s.Players = append(s.Players, &ddb.Player{Id: record.UserId})
	}

	err := sessionRepo.Put(ctx, s)
	if err != nil {
		return err
	}
	s.Broadcast(ctx, fmt.Sprint(record.UserId, " has joined the game."))

	return nil
}

func leaveEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	sessionRepo := ddb.NewSessionRepository()

	s.Connections = slices.DeleteFunc(s.Connections, func(u ddb.Connection) bool { return u.UserId == record.UserId })
	if len(s.Connections) == 0 {
		return sessionRepo.Delete(ctx, s.Id)
	}

	leaveIndex := slices.IndexFunc(s.Players, func(p *ddb.Player) bool { return p.Id == record.UserId })
	if s.Playing() && leaveIndex != -1 {
		if s.CoinToss%2 == 1 {
			if leaveIndex == 0 {
				s.GameTable.Result = "blue resign(leave)"
			} else {
				s.GameTable.Result = "orange resign(leave)"
			}
		} else {
			if leaveIndex == 0 {
				s.GameTable.Result = "orange resign(leave)"
			} else {
				s.GameTable.Result = "blue resign(leave)"
			}
		}
		err := s.ProcessGameResult(ctx, (int(s.CoinToss)+leaveIndex+1)%2)
		if err != nil {
			return err
		}

		ids := []string{s.Players[0].Id, s.Players[1].Id}
		if s.CoinToss%2 == 1 {
			slices.Reverse(ids)
		}
		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[0].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[1].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
	}

	s.Players = slices.DeleteFunc(s.Players, func(u *ddb.Player) bool { return u.Id == record.UserId })

	err := sessionRepo.Put(ctx, s)
	if err != nil {
		return err
	}
	s.Broadcast(ctx, fmt.Sprint(record.UserId, " has left the game."))

	return nil
}

func slotEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	sessionRepo := ddb.NewSessionRepository()

	if s.Playing() {
		return nil
	} else if slices.ContainsFunc(s.Players, func(u *ddb.Player) bool { return u.Id == record.UserId }) {
		s.Players = slices.DeleteFunc(s.Players, func(u *ddb.Player) bool { return u.Id == record.UserId })
	} else if len(s.Players) < 2 {
		s.Players = append(s.Players, &ddb.Player{Id: record.UserId})
	} else {
		return nil
	}

	err := sessionRepo.Put(ctx, s)
	if err != nil {
		return err
	}
	s.Broadcast(ctx, "")

	return nil
}

func startEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	sessionRepo := ddb.NewSessionRepository()

	if record.UserId != s.Players[0].Id || s.Playing() || len(s.Players) != 2 {
		return nil
	}

	s.StartNewGame()

	err := sessionRepo.Put(ctx, s)
	if err != nil {
		return err
	}
	s.Broadcast(ctx, fmt.Sprint("Game start. ", s.CurrentTurnPlayer().Id, " plays first."))

	return nil
}

func gameEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	sessionRepo := ddb.NewSessionRepository()

	if !s.Playing() || s.CurrentTurnPlayer().Id != record.UserId {
		return nil
	}

	if !record.Resign && !s.GameTable.Playable(record.Move) {
		return nil
	}

	now := time.Now().UnixMilli()
	s.CurrentTurnPlayer().RemainingTime -= now - s.LastMoveTime
	s.LastMoveTime = now

	if s.CurrentTurnPlayer().RemainingTime <= 0 || record.Resign {
		color := ""
		if len(s.GameTable.Record)%2 == 0 {
			color = "blue"
		} else {
			color = "orange"
		}
		s.GameTable.Result = fmt.Sprint(color, " resign")

		err := s.ProcessGameResult(ctx, (len(s.GameTable.Record)+int(s.CoinToss))%2)
		if err != nil {
			return err
		}

		ids := []string{s.Players[0].Id, s.Players[1].Id}
		if s.CoinToss%2 == 1 {
			slices.Reverse(ids)
		}

		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[0].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[1].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
		sugarlogger.GetSugar().Info("game finish ", s.GameTable.Result)

		err = sessionRepo.Put(ctx, s)
		if err != nil {
			return err
		}
		s.Broadcast(ctx, s.GameTable.Result)

		return nil
	}

	winner := s.GameTable.MakeMove(record.Move)
	if winner != -1 {
		ids := []string{s.Players[0].Id, s.Players[1].Id}
		if s.CoinToss%2 == 1 {
			slices.Reverse(ids)
		}
		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[0].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
		ddb.NewRecordRepository().Put(ctx, ddb.Record{
			PlayerId:  s.Players[1].Id,
			Time:      time.Now().Format(time.RFC3339),
			PlayersId: ids,
			GameTable: *s.GameTable,
		})
		sugarlogger.GetSugar().Info("game finish ", s.GameTable.Result)

		err := s.ProcessGameResult(ctx, int(s.CoinToss)%2)
		if err != nil {
			return err
		}
		s.Broadcast(ctx, s.GameTable.Result)
	}

	err := sessionRepo.Put(ctx, s)
	if err != nil {
		return err
	}
	s.Broadcast(ctx, "")

	return nil
}

func kickEvent(ctx context.Context, record sqs.Record, s ddb.GameSession) error {
	if !s.Playing() || s.CurrentTurnPlayer().RemainingTime > time.Now().UnixMilli()-s.LastMoveTime {
		return nil
	}

	kickedIndex := slices.IndexFunc(s.Connections, func(u ddb.Connection) bool { return u.UserId == s.CurrentTurnPlayer().Id })
	s.Broadcast(ctx, fmt.Sprint(s.CurrentTurnPlayer().Id, " kicked(timeout)"))
	ws.DeleteWebSocket(ctx, s.Connections[kickedIndex].Id)
	// trigger leave event

	return nil
}

func handler(ctx context.Context, req events.SQSEvent) {
	for _, record := range req.Records {
		r := sqs.Record{}
		json.Unmarshal([]byte(record.Body), &r)
		if r.UserId == "" {
			continue
		}

		sugar := sugarlogger.GetSugar()

		sessionRepo := ddb.NewSessionRepository()
		l, err := sessionRepo.Get(ctx, r.GameTableId)
		if err != nil {
			sugar.Warn(err)
			ws.DeleteWebSocket(ctx, r.Id)
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
			sugar.Warn(err)
			continue
		}
	}
}

func main() {
	lambda.Start(handler)
}
