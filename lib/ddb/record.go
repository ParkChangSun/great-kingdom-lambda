package ddb

import (
	"context"
	"great-kingdom-lambda/lib/game"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Record struct {
	PlayerId  string
	Time      string
	PlayersId []string
	game.GameTable
}

type RecordRepository struct {
	client *dynamodb.Client
}

func NewRecordRepository() *RecordRepository {
	return &RecordRepository{client: _client}
}

func (r *RecordRepository) Put(ctx context.Context, record Record) error {
	data, _ := attributevalue.MarshalMap(record)
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.RECORD_DYNAMODB),
		Item:      data,
	})
	return err
}

type RecordQuery struct {
	Time    string
	Records []Record
}

func (r *RecordRepository) Query(ctx context.Context, playerId string, start string) (RecordQuery, error) {
	key := expression.KeyEqual(expression.Key("PlayerId"), expression.Value(playerId))
	expr, _ := expression.NewBuilder().WithKeyCondition(key).Build()
	var startKey map[string]types.AttributeValue
	if start != "" {
		startKey, _ = attributevalue.MarshalMap(struct{ Time string }{Time: start})
	}

	q, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(vars.RECORD_DYNAMODB),
		ExclusiveStartKey:         startKey,
		ScanIndexForward:          aws.Bool(false),
		Limit:                     aws.Int32(10),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return RecordQuery{}, err
	}

	lastKey := struct{ Time string }{}
	attributevalue.UnmarshalMap(q.LastEvaluatedKey, &lastKey)
	records := []Record{}
	attributevalue.UnmarshalListOfMaps(q.Items, &records)
	return RecordQuery{Time: lastKey.Time, Records: records}, nil
}
