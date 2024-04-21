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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

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
			gameSession.BroadCastChat(ctx, "", fmt.Sprint(msg.UserId, " has joined the game."))

			gameSession.BroadCastGame(ctx)

		case game.LEAVEEVENT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			_, err := dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
				Key:       game.GetConnectionDynamoDBKey(msg.ConnectionId),
			})
			if err != nil {
				return err
			}

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			if gameSession.Game.Playing && slices.ContainsFunc(gameSession.Players, func(u game.User) bool { return u.ConnectionId == msg.ConnectionId }) {
				w := slices.IndexFunc(gameSession.Players, func(u game.User) bool { return u.ConnectionId != msg.ConnectionId })
				winner, _ := game.GetUser(ctx, gameSession.Players[w].UserId)
				winner.W++
				err = winner.UpdateRecord(ctx)
				if err != nil {
					return err
				}

				l := slices.IndexFunc(gameSession.Players, func(u game.User) bool { return u.ConnectionId == msg.ConnectionId })
				loser, _ := game.GetUser(ctx, gameSession.Players[l].UserId)
				loser.W++
				err = winner.UpdateRecord(ctx)
				if err != nil {
					return err
				}

				gameSession.Game.Playing = false
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					return err
				}

				gameSession.BroadCastChat(ctx, "", fmt.Sprint(gameSession.Players[l].UserId, "lost."))
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
				gameSession.BroadCastChat(ctx, "", fmt.Sprintf(msg.ConnectionId, " has left the game."))
			}

		case game.GAMEEVENT:
			// gamestartevent gamemoveevent gamefinishevent
			msg := game.GameMoveSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			if msg.Move.Start {
				if msg.ConnectionId == gameSession.Players[0].ConnectionId && !gameSession.Game.Playing && len(gameSession.Players) == 2 {
					gameSession.StartNewGame(gameSession.Players[0].UserId, gameSession.Players[1].UserId)
					err = gameSession.UpdateGame(ctx)
					if err != nil {
						return err
					}
					gameSession.BroadCastGame(ctx)
					return nil
				} else {
					return nil
				}
			}

			if !gameSession.Game.Playing || gameSession.Game.PlayersId[(gameSession.Game.Turn-1)%2] != msg.UserId {
				return nil
			}

			if msg.Move.Pass {
				if gameSession.Game.Pass() {
					var c string
					if b, o := gameSession.Game.CountTerritory(); b > o {
						c = fmt.Sprint("Game over. ", gameSession.Game.PlayersId[0], " won.")
					} else if b < o {
						c = fmt.Sprint("Game over. ", gameSession.Game.PlayersId[1], " won.")
					} else {
						c = "Game over. Draw."
					}
					gameSession.BroadCastGame(ctx)
					gameSession.BroadCastChat(ctx, "", c)
				} else {
					gameSession.BroadCastGame(ctx)
				}
			} else {
				finished, err := gameSession.Game.Move(msg.Move.Point)
				if err != nil {
					return err
				}

				if finished {
					gameSession.BroadCastGame(ctx)
					gameSession.BroadCastChat(ctx, "", fmt.Sprint("Game over. ", gameSession.Game.PlayersId[gameSession.Game.Turn%2], " won."))
				} else {
					gameSession.BroadCastGame(ctx)
				}
			}

			err = gameSession.UpdateGame(ctx)
			if err != nil {
				return err
			}
			return nil

		case game.CHATEVENT:
			msg := game.GameChatSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			gameSession.BroadCastChat(ctx, msg.UserId, msg.Chat)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
