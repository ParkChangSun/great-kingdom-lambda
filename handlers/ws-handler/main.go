package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

func handler(ctx context.Context, req events.SQSEvent) {
	cfg, _ := config.LoadDefaultConfig(ctx)
	dbClient := dynamodb.NewFromConfig(cfg)

	for _, record := range req.Records {
		switch aws.ToString(record.MessageAttributes["EventType"].StringValue) {
		case game.JOINEVENT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			item, _ := attributevalue.MarshalMap(msg)
			_, err := dbClient.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")), Item: item})
			if err != nil {
				log.Print(err)
				continue
			}

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				log.Print(err)
				continue
			}

			gameSession.CurrentConnections = append(gameSession.CurrentConnections, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			if len(gameSession.Players) < 2 {
				gameSession.Players = append(gameSession.Players, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			}
			err = gameSession.UpdatePlayers(ctx)
			if err != nil {
				log.Print(err)
				continue
			}
			gameSession.BroadCastUser(ctx)
			gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " has joined the game."))

			// for joined user
			gameSession.BroadCastGame(ctx)

		case game.LEAVEEVENT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				log.Print(err)
				continue
			}

			if gameSession.Game.Playing && slices.ContainsFunc(gameSession.Players, func(u game.User) bool { return u.ConnectionId == msg.ConnectionId }) {
				if gameSession.Game.PlayersId[0] == msg.UserId {
					err = gameSession.UpdateGameResult(ctx, 1)
				} else {
					err = gameSession.UpdateGameResult(ctx, 0)
				}
				if err != nil {
					log.Print(err)
					continue
				}
				gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " lost."))

				gameSession.Game.Playing = false
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					log.Print(err)
					continue
				}
				gameSession.BroadCastGame(ctx)
			}

			gameSession.CurrentConnections = slices.DeleteFunc(gameSession.CurrentConnections, func(u game.User) bool {
				return u.ConnectionId == msg.ConnectionId
			})
			gameSession.Players = slices.DeleteFunc(gameSession.Players, func(u game.User) bool {
				return u.ConnectionId == msg.ConnectionId
			})

			if len(gameSession.Players) == 0 && len(gameSession.CurrentConnections) == 0 {
				_, err = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
					TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
					Key:       game.GetGameSessionDynamoDBKey(msg.GameSessionId),
				})
				if err != nil {
					log.Print(err)
					continue
				}
			} else {
				err = gameSession.UpdatePlayers(ctx)
				if err != nil {
					log.Print(err)
					continue
				}
				gameSession.BroadCastUser(ctx)
				gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " has left the game."))
			}

		case game.GAMEEVENT:
			msg := game.GameMoveSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				log.Print(err)
				continue
			}

			if msg.Move.Start && msg.ConnectionId == gameSession.Players[0].ConnectionId && len(gameSession.Players) == 2 && !gameSession.Game.Playing {
				gameSession.StartNewGame(gameSession.Players[0].UserId, gameSession.Players[1].UserId)
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					log.Print(err)
					continue
				}
				gameSession.BroadCastGame(ctx)
				gameSession.BroadCastChat(ctx, fmt.Sprint("Game start. ", gameSession.Game.PlayersId[0], " plays first."))

				continue
			}

			if !gameSession.Game.Playing || gameSession.Game.PlayersId[(gameSession.Game.Turn-1)%2] != msg.UserId {
				continue
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
					log.Print(err)
					continue
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
				log.Print(err)
				continue
			}
			gameSession.BroadCastGame(ctx)

		case game.SLOTEVENT:
			msg := game.GameChatSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				log.Print(err)
				continue
			}

			gameSession.BroadCastUser(ctx)

		case game.CHATEVENT:
			msg := game.GameChatSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				log.Print(err)
				continue
			}

			gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " : ", msg.Chat))

		case game.GLOBALCHAT:
			msg := game.ConnectionDDBItem{}
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

		default:
			continue
		}
	}
}

func main() {
	lambda.Start(handler)
}
