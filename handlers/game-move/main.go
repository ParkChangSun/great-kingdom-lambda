package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type obj struct {
	Id string `dynamodbav:"id"`
}

func handler(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	client := dynamodb.NewFromConfig(cfg)

	name := os.Getenv("STATEDYNAMODB")
	t, _ := attributevalue.MarshalMap(obj{"foo"})
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(name),
		Item:                t,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	v, _ := attributevalue.Marshal("bar")
	i, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       map[string]types.AttributeValue{"id": v},
		TableName: aws.String(name),
	})

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var o obj
	attributevalue.UnmarshalMap(i.Item, &o)

	return events.APIGatewayProxyResponse{Body: o.Id, StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
