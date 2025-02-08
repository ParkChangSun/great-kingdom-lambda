package ddb

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ConnectionDDBItem struct {
	ConnectionId  string
	Timestamp     int64
	GameSessionId string
	UserId        string
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
	if err != nil {
		return ConnectionDDBItem{}, err
	}
	return item, nil
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

	record := ConnectionDDBItem{}
	attributevalue.UnmarshalMap(query.Item, &record)
	return record, nil
}

type UserDDBItem struct {
	UserUUID     string
	UserId       string
	PasswordHash string `json:"-"`
	RefreshToken string `json:"-"`
	W, L         int
}

func GetUser(ctx context.Context, userId string) (UserDDBItem, error) {
	item := UserDDBItem{}

	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: userId})
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       k,
	})
	if err != nil {
		return item, err
	}

	if len(query.Item) == 0 {
		return item, fmt.Errorf("getuser: user not found")
	}

	attributevalue.UnmarshalMap(query.Item, &item)
	return item, nil
}

func GetUserByRefreshToken(ctx context.Context, token string) (UserDDBItem, error) {
	key := expression.KeyEqual(expression.Key("RefreshToken"), expression.Value(token))
	expr, _ := expression.NewBuilder().WithKeyCondition(key).Build()
	query, err := client(ctx).Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("USER_DYNAMODB")),
		IndexName:                 aws.String("RefreshToken"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return UserDDBItem{}, err
	}

	if query.Count == 0 {
		return UserDDBItem{}, fmt.Errorf("getuserbyrefreshtoken: user not found")
	}

	item := []UserDDBItem{}
	attributevalue.UnmarshalListOfMaps(query.Items, &item)
	return item[0], nil
}

func (u UserDDBItem) SyncRecord(ctx context.Context) error {
	update := expression.Set(
		expression.Name("W"),
		expression.Value(u.W),
	).Set(
		expression.Name("L"),
		expression.Value(u.L),
	)
	// condition := expression.AttributeExists(expression.Name("UserId"))
	// expr, _ := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()
	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: u.UserId})
	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       k,
		// ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (u UserDDBItem) SyncRefreshToken(ctx context.Context) error {
	update := expression.Set(
		expression.Name("RefreshToken"),
		expression.Value(u.RefreshToken),
	)
	// condition := expression.AttributeExists(expression.Name("UserId"))
	// expr, _ := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()
	k, _ := attributevalue.MarshalMap(struct{ UserId string }{UserId: u.UserId})
	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       k,
		// ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	if err != nil {
		return err
	}
	return nil
}
