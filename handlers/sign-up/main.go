package main

import (
	"context"
	"encoding/json"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	body := struct {
		Id       string
		Password string
	}{}
	json.Unmarshal([]byte(req.Body), &body)

	// getitem 값없을때 에러인가? 어쨌든 중복 처리
	// 72 byte limit 처리 최소값 한 6자리 정도 처리

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	data, _ := attributevalue.MarshalMap(game.UserDDBItem{
		UserUUID:     uuid.NewString(),
		UserId:       body.Id,
		PasswordHash: string(hash),
	})
	_, err = dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Item:      data,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 201}, nil
}

func main() {
	lambda.Start(handler)
}
