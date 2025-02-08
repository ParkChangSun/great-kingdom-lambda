package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sam-app/awsutils"
	"sam-app/ddb"
	"sam-app/game"
	"slices"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func joinEvent(ctx context.Context, record events.SQSMessage) error {
	conn := ddb.ConnectionDDBItem{}
	json.Unmarshal([]byte(record.Body), &conn)

	err := ddb.PutConnInPool(ctx, conn)
	if err != nil {
		return err
	}

	gameSession, err := ddb.GetGameSession(ctx, conn.GameSessionId)
	if err != nil {
		return err
	}

	userConn := ddb.User{ConnectionId: conn.ConnectionId, UserId: conn.UserId}
	gameSession.CurrentConnections = append(gameSession.CurrentConnections, userConn)
	if len(gameSession.Players) < 2 {
		gameSession.Players = append(gameSession.Players, userConn)
	}
	err = gameSession.UpdatePlayers(ctx)
	if err != nil {
		return err
	}

	gameSession.BroadCastUser(ctx)
	gameSession.BroadCastGame(ctx)
	gameSession.BroadCastChat(ctx, fmt.Sprint(conn.UserId, " has joined the game."))
	return nil
}

func leaveEvent(ctx context.Context, record events.SQSMessage, dbClient *dynamodb.Client) error {
	conn := ddb.ConnectionDDBItem{}
	json.Unmarshal([]byte(record.Body), &conn)

	gameSession, err := ddb.GetGameSession(ctx, conn.GameSessionId)
	if err != nil {
		return err
	}

	if gameSession.Game.Playing && slices.ContainsFunc(gameSession.Players, func(u ddb.User) bool { return u.ConnectionId == conn.ConnectionId }) {
		if gameSession.Game.PlayersId[0] == conn.UserId {
			err = gameSession.UpdateGameResult(ctx, 1)
		} else {
			err = gameSession.UpdateGameResult(ctx, 0)
		}
		if err != nil {
			return err
		}
		gameSession.BroadCastChat(ctx, fmt.Sprint(conn.UserId, " lost."))

		gameSession.Game.Playing = false
		err = gameSession.UpdateGame(ctx)
		if err != nil {
			return err
		}
		gameSession.BroadCastGame(ctx)
	}

	gameSession.CurrentConnections = slices.DeleteFunc(gameSession.CurrentConnections, func(u ddb.User) bool {
		return u.ConnectionId == conn.ConnectionId
	})
	gameSession.Players = slices.DeleteFunc(gameSession.Players, func(u ddb.User) bool {
		return u.ConnectionId == conn.ConnectionId
	})

	if len(gameSession.Players) == 0 && len(gameSession.CurrentConnections) == 0 {
		key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{conn.GameSessionId})
		_, err = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
			Key:       key,
		})
		if err != nil {
			return err
		}
	} else {
		err = gameSession.UpdatePlayers(ctx)
		if err != nil {
			return err
		}
		gameSession.BroadCastUser(ctx)
		gameSession.BroadCastChat(ctx, fmt.Sprint(conn.UserId, " has left the game."))
	}
	return nil
}

func chatEvent(ctx context.Context, record events.SQSMessage) error {
	msg := awsutils.GameChatSQSRecord{}
	json.Unmarshal([]byte(record.Body), &msg)

	gameSession, err := ddb.GetGameSession(ctx, msg.GameSessionId)
	if err != nil {
		return err
	}

	gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " : ", msg.Chat))
	return nil
}

func slotEvent(ctx context.Context, record events.SQSMessage) error {
	msg := awsutils.GameChatSQSRecord{}
	json.Unmarshal([]byte(record.Body), &msg)

	gameSession, err := ddb.GetGameSession(ctx, msg.GameSessionId)
	if err != nil {
		return err
	}

	if slices.ContainsFunc(gameSession.Players, func(u ddb.User) bool { return u.UserId == msg.UserId }) {
		gameSession.Players = slices.DeleteFunc(gameSession.Players, func(u ddb.User) bool { return u.UserId == msg.UserId })
	} else if len(gameSession.Players) < 2 {
		gameSession.Players = append(gameSession.Players, ddb.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
	} else {
		return fmt.Errorf("user error")
	}

	err = gameSession.UpdatePlayers(ctx)
	if err != nil {
		return err
	}
	gameSession.BroadCastUser(ctx)
	return nil
}

func gameEvent(ctx context.Context, record events.SQSMessage) error {
	msg := awsutils.GameMoveSQSRecord{}
	json.Unmarshal([]byte(record.Body), &msg)

	gameSession, err := ddb.GetGameSession(ctx, msg.GameSessionId)
	if err != nil {
		return err
	}

	if msg.Move.Start && msg.ConnectionId == gameSession.Players[0].ConnectionId && len(gameSession.Players) == 2 && !gameSession.Game.Playing {
		gameSession.StartNewGame(gameSession.Players[0].UserId, gameSession.Players[1].UserId)
		err = gameSession.UpdateGame(ctx)
		if err != nil {
			return err
		}
		gameSession.BroadCastGame(ctx)
		gameSession.BroadCastChat(ctx, fmt.Sprint("Game start. ", gameSession.Game.PlayersId[0], " plays first."))

		return nil
	}

	if !gameSession.Game.Playing || gameSession.Game.PlayersId[(gameSession.Game.Turn-1)%2] != msg.UserId {
		return nil
	}

	if msg.Move.Pass {
		if gameSession.Game.Pass() {
			if b, o := gameSession.Game.CountTerritory(); b >= o+3 {
				gameSession.UpdateGameResult(ctx, 0)
				gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", b, ":", o, " ", gameSession.Game.PlayersId[0], " won."))
			} else {
				gameSession.UpdateGameResult(ctx, 1)
				gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", b, ":", o, " ", gameSession.Game.PlayersId[1], " won."))
			}
		}
	} else {
		sieged, err := gameSession.Game.Move(msg.Move.Point)
		if err != nil {
			return nil
		}
		if sieged {
			gameSession.UpdateGameResult(ctx, (gameSession.Game.Turn-1)%2)
			gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[(gameSession.Game.Turn-1)%2], " won."))
		} else {
			movable := false
			for _, r := range gameSession.Game.Board {
				for _, c := range r {
					if c == game.EmptyCell {
						movable = true
					}
				}
			}
			if !movable {
				gameSession.Game.Playing = false
				if b, o := gameSession.Game.CountTerritory(); b >= o+3 {
					gameSession.UpdateGameResult(ctx, 0)
					gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", b, ":", o, " ", gameSession.Game.PlayersId[0], " won."))
				} else {
					gameSession.UpdateGameResult(ctx, 1)
					gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", b, ":", o, " ", gameSession.Game.PlayersId[1], " won."))
				}
			}
		}
	}
	err = gameSession.UpdateGame(ctx)
	if err != nil {
		return err
	}
	gameSession.BroadCastGame(ctx)
	return nil
}

func handler(ctx context.Context, req events.SQSEvent) {
	cfg, _ := config.LoadDefaultConfig(ctx)
	dbClient := dynamodb.NewFromConfig(cfg)

	for _, record := range req.Records {
		var err error
		switch aws.ToString(record.MessageAttributes["EventType"].StringValue) {
		case awsutils.JOINEVENT:
			err = joinEvent(ctx, record)

		case awsutils.LEAVEEVENT:
			err = leaveEvent(ctx, record, dbClient)

		case awsutils.GAMEEVENT:
			err = gameEvent(ctx, record)

		case awsutils.SLOTEVENT:
			err = slotEvent(ctx, record)

		case awsutils.CHATEVENT:
			err = chatEvent(ctx, record)

		case awsutils.GLOBALCHAT:
			msg := ddb.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			chatkey := expression.KeyEqual(expression.Key("ChatName"), expression.Value("globalchat"))
			expr, _ := expression.NewBuilder().WithKeyCondition(chatkey).Build()
			out, err := dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
				TableName:                 aws.String(os.Getenv("GLOBAL_CHAT_DYNAMODB")),
				ScanIndexForward:          aws.Bool(false),
				Limit:                     aws.Int32(50),
				KeyConditionExpression:    expr.KeyCondition(),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
			})
			if err != nil {
				log.Print(err)
				continue
			}

			lastKey := struct{ Timestamp int64 }{}
			attributevalue.UnmarshalMap(out.LastEvaluatedKey, &lastKey)

			lastmsgs := []struct {
				Chat      string
				ChatName  string
				Timestamp int64
			}{}
			attributevalue.UnmarshalListOfMaps(out.Items, &lastmsgs)
			payload := struct {
				EventType string
				Messages  []struct {
					Chat      string
					ChatName  string
					Timestamp int64
				}
				LastScroll int64
			}{
				EventType:  "lastchat",
				Messages:   lastmsgs,
				LastScroll: lastKey.Timestamp,
			}
			b, _ := json.Marshal(payload)

			apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
				o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
			}).PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: aws.String(msg.ConnectionId),
				Data:         b,
			})
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
