package main

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
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

	user, _ := game.GetUser(ctx, body.Id)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }
	if user.UserId != "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "id exists", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}

	num := regexp.MustCompile(`.*[0-9]`)
	eng := regexp.MustCompile(`.*[a-zA-Z]`)
	bytelen := regexp.MustCompile(`^.{6,30}$`)
	if !num.Match([]byte(body.Password)) || !eng.Match([]byte(body.Password)) || !bytelen.Match([]byte(body.Password)) {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "password invalid", Headers: map[string]string{"Access-Control-Allow-Origin": "*"}}, nil
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

	data, _ := attributevalue.MarshalMap(game.UserDDBItem{
		UserUUID:     uuid.NewString(),
		UserId:       body.Id,
		PasswordHash: string(hash),
		RefreshToken: "logout",
	})
	_, err = dynamodb.NewFromConfig(cfg).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Item:      data,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    game.DefaultCORSHeaders,
	}, nil
}

func main() {
	lambda.Start(handler)
}
