package ddb

import (
	"context"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UserDDBItem struct {
	UserId       string
	PasswordHash string `json:"-"`
	RefreshToken string `json:"-"`
	W, L         int
	RecentGames  []RecentGame
}

type RecentGame struct {
	BlueId, OrangeId, WinnerId string
	CreatedDate                string
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
		TableName: aws.String(vars.USER_DYNAMODB),
		Item:      data,
	})
	return err
}

func GetUser(ctx context.Context, userId string) (UserDDBItem, error) {
	query, err := client(ctx).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(vars.USER_DYNAMODB),
		Key:       UserDDBItemKey(userId),
	})
	if err != nil {
		return UserDDBItem{}, err
	}
	if query.Item == nil {
		return UserDDBItem{}, &ItemNotFoundError{vars.USER_DYNAMODB, userId}
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
		TableName:                 aws.String(vars.USER_DYNAMODB),
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
		TableName:                 aws.String(vars.USER_DYNAMODB),
		Key:                       UserDDBItemKey(u.UserId),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err
}
