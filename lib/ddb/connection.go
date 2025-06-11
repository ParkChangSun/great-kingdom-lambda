package ddb

import (
	"context"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ConnectionDDBItem struct {
	ConnectionId string
	UserId       string
	GameTableId  string
	CreatedDate  string
	Authorized   bool
}

func PutConnInPool(ctx context.Context, conn ConnectionDDBItem) error {
	item, _ := attributevalue.MarshalMap(conn)

	_, err := client(ctx).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.CONNECTION_DYNAMODB),
		Item:      item,
	})

	return err
}

func DeleteConnInPool(ctx context.Context, connId string) (ConnectionDDBItem, error) {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{connId})

	out, err := client(ctx).DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:    aws.String(vars.CONNECTION_DYNAMODB),
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
		TableName: aws.String(vars.CONNECTION_DYNAMODB),
		Key:       key,
	})
	if err != nil {
		return ConnectionDDBItem{}, err
	}
	if query.Item == nil {
		return ConnectionDDBItem{}, &ItemNotFoundError{vars.CONNECTION_DYNAMODB, connectionId}
	}

	record := ConnectionDDBItem{}
	attributevalue.UnmarshalMap(query.Item, &record)
	return record, nil
}

func (c ConnectionDDBItem) UpdateUserId(ctx context.Context) error {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{c.ConnectionId})
	update := expression.Set(
		expression.Name("UserId"),
		expression.Value(c.UserId),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()
	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(vars.CONNECTION_DYNAMODB),
		Key:                       key,
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	return err
}

func (c ConnectionDDBItem) UpdateAuthorized(ctx context.Context) error {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{c.ConnectionId})
	update := expression.Set(
		expression.Name("Authorized"),
		expression.Value(c.Authorized),
	)
	expr, _ := expression.NewBuilder().WithUpdate(update).Build()
	_, err := client(ctx).UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(vars.CONNECTION_DYNAMODB),
		Key:                       key,
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	return err
}

func QueryGlobalChat(ctx context.Context) ([]ConnectionDDBItem, error) {
	key := expression.KeyEqual(expression.Key("GameTableId"), expression.Value("globalchat"))
	expr, _ := expression.NewBuilder().WithKeyCondition(key).Build()
	q, err := client(ctx).Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(vars.CONNECTION_DYNAMODB),
		IndexName:                 aws.String("globalchat"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, err
	}

	out := []ConnectionDDBItem{}
	err = attributevalue.UnmarshalListOfMaps(q.Items, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
