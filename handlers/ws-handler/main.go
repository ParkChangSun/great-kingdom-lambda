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
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func handler(ctx context.Context, req events.SQSEvent) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	wsClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("API_ENDPOINT"))
	})

	for _, record := range req.Records {
		switch aws.ToString(record.MessageAttributes["EventType"].StringValue) {
		case game.JOINEVENT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			item, _ := attributevalue.MarshalMap(msg)
			_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")), Item: item})
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

			gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
				EventType:          game.JOINEVENT,
				Players:            gameSession.Players,
				CurrentConnections: gameSession.CurrentConnections,
				Chat:               fmt.Sprintf("%s has joined the game.", msg.ConnectionId),
			})

			gameData, _ := json.Marshal(game.ServerToClient{
				EventType: game.GAMEEVENT,
				Game:      gameSession.Game,
			})
			_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: aws.String(msg.ConnectionId),
				Data:         gameData,
			})
			if err != nil {
				log.Print(err)
			}

		case game.LEAVEEVENT:
			msg := game.ConnectionDDBItem{}
			json.Unmarshal([]byte(record.Body), &msg)

			_, err = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
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

				gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
					EventType:          game.LEAVEEVENT,
					Players:            gameSession.Players,
					CurrentConnections: gameSession.CurrentConnections,
					Chat:               fmt.Sprintf("%s has left the game.", msg.ConnectionId),
				})
			}

		case game.GAMEEVENT:
			// gamestartevent gamemoveevent gamefinishevent
			msg := game.GameMoveSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			if msg.Move.Start && msg.ConnectionId == gameSession.Players[0].ConnectionId && !gameSession.Game.Playing {
				gameSession.StartNewGame(gameSession.Players[0].UserId, gameSession.Players[1].UserId)
				err = gameSession.UpdateGame(ctx)
				if err != nil {
					return err
				}
				return nil
			}

			if !gameSession.Game.Playing || gameSession.Game.PlayersId[gameSession.Game.Turn] != msg.UserId {
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
					gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
						EventType: game.GAMEEVENT,
						Game:      gameSession.Game,
						Chat:      c,
					})
				} else {
					gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
						EventType: game.GAMEEVENT,
						Game:      gameSession.Game,
					})
				}
			} else {
				finished, err := gameSession.Game.Move(msg.Move.Point)
				if err != nil {
					return err
				}

				if finished {
					gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
						EventType: game.GAMEEVENT,
						Game:      gameSession.Game,
						Chat:      fmt.Sprint("Game over. ", gameSession.Game.PlayersId[gameSession.Game.Turn%2], " won."),
					})
				} else {
					gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
						EventType: game.GAMEEVENT,
						Game:      gameSession.Game,
					})
				}
			}

			err = gameSession.UpdateGame(ctx)
			if err != nil {
				return err
			}

		case game.CHATEVENT:
			msg := game.GameChatSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			gameSession, err := game.GetGameSession(ctx, msg.GameSessionId)
			if err != nil {
				return err
			}

			gameSession.SendWebSocketMessage(ctx, game.ServerToClient{
				EventType: game.CHATEVENT,
				Chat:      msg.Chat,
			})
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
