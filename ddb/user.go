package ddb

import (
	"context"
	"os"
	"sam-app/game"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ConnectionDDBItem struct {
	UserId        string
	ConnectionId  string
	GameSessionId string
}

func PutConnInPool(ctx context.Context, conn ConnectionDDBItem) error {
	item, _ := attributevalue.MarshalMap(conn)

	_, err := client(ctx).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Item:      item,
	})

	return err
}

func DeleteConnInPool(ctx context.Context, connId string) (ConnectionDDBItem, error) {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{connId})

	out, err := client(ctx).DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:    aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Key:          key,
		ReturnValues: "ALL_OLD",
	})
	if err != nil {
		return ConnectionDDBItem{}, err
	}

	item := ConnectionDDBItem{}
	err = attributevalue.UnmarshalMap(out.Attributes, &item)
	return item, err
}

func GetConnection(ctx context.Context, connectionId string) (ConnectionDDBItem, error) {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{connectionId})
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("CONNECTION_DYNAMODB")),
		Key:       key,
	})
	if err != nil {
		return ConnectionDDBItem{}, err
	}
	if query.Item == nil {
		return ConnectionDDBItem{}, ErrItemNotFound
	}

	record := ConnectionDDBItem{}
	attributevalue.UnmarshalMap(query.Item, &record)
	return record, nil
}

type Record struct {
	EventType string
	ConnectionDDBItem

	Chat string
	Move game.Move

	Timestamp int64
}

type UserDDBItem struct {
	UserId       string
	PasswordHash string `json:"-"`
	RefreshToken string `json:"-"`
	W, L         int
	RecentGames  []RecentGame
}

type RecentGame struct {
	BlueId, OrangeId, WinnerId string
}

func UserDDBItemKey(userId string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: userId})
	return k
}

func PutUser(ctx context.Context, id, pwh string) error {
	data, _ := attributevalue.MarshalMap(UserDDBItem{
		UserId:       id,
		PasswordHash: pwh,
		RefreshToken: "",
		RecentGames:  []RecentGame{},
	})
	_, err := client(ctx).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Item:      data,
	})
	return err
}

func GetUser(ctx context.Context, userId string) (UserDDBItem, error) {
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       UserDDBItemKey(userId),
	})
	if err != nil {
		return UserDDBItem{}, err
	}
	if query.Item == nil {
		return UserDDBItem{}, ErrItemNotFound
	}

	item := UserDDBItem{}
	err = attributevalue.UnmarshalMap(query.Item, &item)
	return item, err
}

func (u UserDDBItem) SyncRecord(ctx context.Context) error {
	update := expression.Set(
		expression.Name("W"),
		expression.Value(u.W),
	).Set(
		expression.Name("L"),
		expression.Value(u.L),
	).Set(
		expression.Name("RecentGames"),
		expression.Value(u.RecentGames),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("USER_DYNAMODB")),
		Key:                       UserDDBItemKey(u.UserId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err
}

func (u UserDDBItem) SyncRefreshToken(ctx context.Context) error {
	update := expression.Set(
		expression.Name("RefreshToken"),
		expression.Value(u.RefreshToken),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()

	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("USER_DYNAMODB")),
		Key:                       UserDDBItemKey(u.UserId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err
}
