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

type Connection struct {
	Id          string
	UserId      string
	GameTableId string
	CreatedDate string
	Authorized  bool
}

type ConnectionRepository struct {
	client *dynamodb.Client
}

func NewConnectionRepository() *ConnectionRepository {
	return &ConnectionRepository{client: _client}
}

func ConnectionKey(id string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ Id string }{Id: id})
	return k
}

func (r *ConnectionRepository) Get(ctx context.Context, id string) (Connection, error) {
	query, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(vars.CONNECTION_DYNAMODB),
		Key:       ConnectionKey(id),
	})
	if err != nil {
		return Connection{}, err
	}

	if query.Item == nil {
		return Connection{}, &ItemNotFoundError{vars.CONNECTION_DYNAMODB, id}
	}

	item := Connection{}
	attributevalue.UnmarshalMap(query.Item, &item)
	return item, nil
}

func (r *ConnectionRepository) Put(ctx context.Context, conn Connection) error {
	item, _ := attributevalue.MarshalMap(conn)

	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.CONNECTION_DYNAMODB),
		Item:      item,
	})

	return err
}

func (r *ConnectionRepository) Delete(ctx context.Context, id string) (Connection, error) {
	out, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:    aws.String(vars.CONNECTION_DYNAMODB),
		Key:          ConnectionKey(id),
		ReturnValues: "ALL_OLD",
	})
	if err != nil {
		return Connection{}, err
	}

	item := Connection{}
	err = attributevalue.UnmarshalMap(out.Attributes, &item)
	return item, err
}

func (r *ConnectionRepository) Query(ctx context.Context) ([]Connection, error) {
	key := expression.KeyEqual(expression.Key("GameTableId"), expression.Value("globalchat"))
	expr, _ := expression.NewBuilder().WithKeyCondition(key).Build()
	q, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(vars.CONNECTION_DYNAMODB),
		IndexName:                 aws.String("globalchat"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, err
	}

	out := []Connection{}
	attributevalue.UnmarshalListOfMaps(q.Items, &out)
	return out, nil
}
