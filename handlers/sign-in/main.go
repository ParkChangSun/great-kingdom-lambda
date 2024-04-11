package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sam-app/game"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/golang-jwt/jwt/v5"
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

	// getitem 아이템 없으면 에러인가?
	k, _ := attributevalue.MarshalMap(struct{ Id string }{body.Id})
	out, err := dynamodb.NewFromConfig(cfg).GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USER_DYNAMODB")),
		Key:       k,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	item := game.UserDDBItem{}
	attributevalue.UnmarshalMap(out.Item, item)

	err = bcrypt.CompareHashAndPassword([]byte(item.PasswordHash), []byte(body.Password))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	type AuthTokenClaims struct {
		jwt.RegisteredClaims
		UserId string
	}
	t := AuthTokenClaims{}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, t)
	tstr, err := token.SignedString("key")
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers:    map[string]string{"Set-Cookie": fmt.Sprint("GreatKingdomAuth=bearer ", tstr, ";SameSite=None;Secure")},
	}, nil
}

func main() {
	lambda.Start(handler)
}
