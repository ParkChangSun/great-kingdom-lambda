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
		eventType := aws.ToString(record.MessageAttributes["EventType"].StringValue)
		switch eventType {
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

			item, _ = attributevalue.MarshalMap(gameSession)
			dbClient.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")), Item: item})

			joinData, _ := json.Marshal(game.ServerToClient{
				EventType:          game.JOINEVENT,
				Players:            gameSession.Players,
				CurrentConnections: gameSession.CurrentConnections,
				Chat:               fmt.Sprintf("%s has joined the game.", msg.ConnectionId),
			})
			for _, c := range gameSession.CurrentConnections {
				_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
					ConnectionId: aws.String(c.ConnectionId),
					Data:         joinData,
				})
				if err != nil {
					log.Print(err)
				}
			}

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
				item, _ := attributevalue.MarshalMap(gameSession)
				dbClient.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
					Item:      item,
				})

				leaveData, _ := json.Marshal(game.ServerToClient{
					EventType:          game.LEAVEEVENT,
					Players:            gameSession.Players,
					CurrentConnections: gameSession.CurrentConnections,
					Chat:               fmt.Sprintf("%s has left the game.", msg.ConnectionId),
				})
				for _, c := range gameSession.CurrentConnections {
					_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
						ConnectionId: aws.String(c.ConnectionId),
						Data:         leaveData,
					})
					if err != nil {
						log.Print(err)
					}
				}
			}

		case game.GAMEEVENT:
			msg := game.GameMoveSQSRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			out, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
				Key:       game.GetGameSessionDynamoDBKey(msg.GameSessionId),
			})
			if err != nil {
				return err
			}

			gameSession := game.GameSession{}
			attributevalue.UnmarshalMap(out.Item, &gameSession)

			if !gameSession.Game.Playing || gameSession.Players[gameSession.Game.Turn].ConnectionId != msg.ConnectionId {
				return nil
			}

			// game logic
			// check territory, sieged

			// send game event to client
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
