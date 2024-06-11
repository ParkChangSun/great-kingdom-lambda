package main

import (
	"context"
	"encoding/json"
	"fmt"
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

// guess should not return for for loop

func handler(ctx context.Context, req events.SQSEvent) error {
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
				return err
			}

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			gameSession.CurrentConnections = append(gameSession.CurrentConnections, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			if len(gameSession.Players) < 2 {
				gameSession.Players = append(gameSession.Players, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			}
			err = gameSession.UpdatePlayers(ctx)
			if err != nil {
				return err
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
				return err
			}

			if gameSession.Game.Playing && slices.ContainsFunc(gameSession.Players, func(u game.User) bool { return u.ConnectionId == msg.ConnectionId }) {
				if gameSession.Game.PlayersId[0] == msg.UserId {
					err = gameSession.UpdateGameResult(ctx, 1)
				} else {
					err = gameSession.UpdateGameResult(ctx, 0)
				}
				if err != nil {
					return err
				}
				gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " lost."))

				gameSession.Game.Playing = false
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					return err
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
					return err
				}
			} else {
				err = gameSession.UpdatePlayers(ctx)
				if err != nil {
					return err
				}
				gameSession.BroadCastUser(ctx)
				gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " has left the game."))
			}

		case game.GAMEEVENT:
			// gamestartevent gamemoveevent gamefinishevent
			msg := game.GameMoveSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			if msg.Move.Start && msg.ConnectionId == gameSession.Players[0].ConnectionId && !gameSession.Game.Playing && len(gameSession.Players) == 2 {
				gameSession.StartNewGame(gameSession.Players[0].UserId, gameSession.Players[1].UserId)
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					return err
				}
				gameSession.BroadCastGame(ctx)

				return nil
			}

			if !gameSession.Game.Playing || gameSession.Game.PlayersId[(gameSession.Game.Turn-1)%2] != msg.UserId {
				return nil
			}

			if msg.Move.Pass {
				if gameSession.Game.Pass() {
					if b, o := gameSession.Game.CountTerritory(); b > o+2 {
						gameSession.UpdateGameResult(ctx, 0)
						gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[0], " won."))
					} else if b < o+2 {
						gameSession.UpdateGameResult(ctx, 1)
						gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[1], " won."))
					} else {
						gameSession.UpdateGameResult(ctx, -1)
						gameSession.BroadCastChat(ctx, "Game over. Draw.")
					}
				}
			} else {
				sieged, err := gameSession.Game.Move(msg.Move.Point)
				if err != nil {
					return err
				}
				if sieged {
					gameSession.UpdateGameResult(ctx, (gameSession.Game.Turn-1)%2)
					gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[gameSession.Game.Turn%2], " won."))
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
						if b, o := gameSession.Game.CountTerritory(); b > o+2 {
							gameSession.UpdateGameResult(ctx, 0)
							gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[0], " won."))
						} else if b < o+2 {
							gameSession.UpdateGameResult(ctx, 1)
							gameSession.BroadCastChat(ctx, fmt.Sprint("Game over. ", gameSession.Game.PlayersId[1], " won."))
						} else {
							gameSession.UpdateGameResult(ctx, -1)
							gameSession.BroadCastChat(ctx, "Game over. Draw.")
						}
					}
				}
			}
			err = gameSession.UpdateGame(ctx)
			if err != nil {
				return err
			}
			gameSession.BroadCastGame(ctx)

		case game.CHATEVENT:
			msg := game.GameChatSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			gameSession.BroadCastChat(ctx, fmt.Sprint(msg.UserId, " : ", msg.Chat))

		case game.GLOBALCHAT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			chatkey := expression.KeyEqual(expression.Key("ChatName"), expression.Value("globalchat"))
			expr, _ := expression.NewBuilder().WithKeyCondition(chatkey).Build()
			out, err := dynamodb.NewFromConfig(cfg).Query(ctx, &dynamodb.QueryInput{
				TableName: aws.String(os.Getenv("GLOBAL_CHAT_DYNAMODB")),
				// ScanIndexForward: aws.Bool(true),
				KeyConditionExpression:    expr.KeyCondition(),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
			})
			if err != nil {
				return err
			}

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
			}{
				EventType: "lastchat",
				Messages:  lastmsgs,
			}
			b, _ := json.Marshal(payload)

			apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
				o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_ENDPOINT"))
			}).PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: aws.String(msg.ConnectionId),
				Data:         b,
			})

		default:
			return nil
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
