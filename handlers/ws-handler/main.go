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
		log.Print("eventtype:", eventType)
		switch eventType {
		case game.JOINEVENT:
			msg := game.JoinRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			item, _ := attributevalue.MarshalMap(msg)
			_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")), Item: item})
			if err != nil {
				return err
			}

			out, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
				Key:       game.GetGameSessionDynamoDBKey(msg.GameSessionId),
			})
			if err != nil {
				return err
			}

			gameSession := game.GameSession{}
			attributevalue.UnmarshalMap(out.Item, &gameSession)

			gameSession.CurrentConnections = append(gameSession.CurrentConnections, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			if len(gameSession.Players) < 2 {
				gameSession.Players = append(gameSession.Players, game.User{ConnectionId: msg.ConnectionId, UserId: msg.UserId})
			}

			item, _ = attributevalue.MarshalMap(gameSession)
			dbClient.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")), Item: item})

			data, _ := json.Marshal(struct {
				EventType string
				Chat      string
			}{
				game.CHATEVENT,
				fmt.Sprintf("%s has joined the game.", msg.ConnectionId),
			})
			for _, c := range gameSession.CurrentConnections {
				_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
					ConnectionId: aws.String(c.ConnectionId),
					Data:         data,
				})
				if err != nil {
					log.Print(err)
				}
			}
		case game.LEAVEEVENT:
			msg := game.JoinRecord{}
			json.Unmarshal([]byte(record.Body), &msg)

			_, err = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
				Key:       game.GetConnectionDynamoDBKey(msg.ConnectionId),
			})
			if err != nil {
				return err
			}

			out, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB")),
				Key:       game.GetGameSessionDynamoDBKey(msg.GameSessionId),
			})
			if err != nil {
				return err
			}

			gameSession := game.GameSession{}
			attributevalue.UnmarshalMap(out.Item, &gameSession)

			log.Printf("%+v\n", gameSession)

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

				data := struct {
					EventType string
					Chat      string
				}{game.CHATEVENT, fmt.Sprintf("%s has left the game.", msg.ConnectionId)}
				databyte, _ := json.Marshal(data)
				for _, c := range gameSession.CurrentConnections {
					_, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
						ConnectionId: aws.String(c.ConnectionId),
						Data:         []byte(databyte),
					})
					if err != nil {
						log.Print(err)
					}
				}
			}
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}

// update := expression.Set(expression.Name("GameSessionId"), expression.Value(sessionId))
// 	expr, err := expression.NewBuilder().WithUpdate(update).Build()

// 	dbClient := dynamodb.NewFromConfig(cfg)
// 	out, err := dbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
// 		TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
// 	})
// 	if err != nil {
// 		return events.APIGatewayProxyResponse{}, err
// 	}

// 	msgbody, err := json.Marshal(game.Chat{
// 		Timestamp:     time.Now().UnixMilli(),
// 		ConnectionId:  req.RequestContext.ConnectionID,
// 		GameSessionId: sessionId,
// 		Message:       fmt.Sprintf("%s has joined.", req.RequestContext.ConnectionID),
// 	})
// 	if err != nil {
// 		return events.APIGatewayProxyResponse{}, err
// 	}

// dbClient := dynamodb.NewFromConfig(cfg)

// dbItem, _ := attributevalue.MarshalMap(struct {
// 	ConnectionId string
// }{req.RequestContext.ConnectionID})

// out, err := dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
// 	TableName:    aws.String(os.Getenv("CONNECTION_DYNAMODB")),
// 	Key:          dbItem,
// 	ReturnValues: types.ReturnValueAllOld,
// })
// if err != nil {
// 	return events.APIGatewayProxyResponse{}, err
// }

// outobj := game.WebSocketClient{}
// attributevalue.UnmarshalMap(out.Attributes, &outobj)

// msgbody, err := json.Marshal(chat.Chat{
// 	Timestamp:     time.Now().UnixMilli(),
// 	ConnectionId:  req.RequestContext.ConnectionID,
// 	GameSessionId: outobj.GameSessionId,
// 	Message:       fmt.Sprintf("%s has left.", req.RequestContext.ConnectionID),
// })
// if err != nil {
// 	return events.APIGatewayProxyResponse{}, err
// }
