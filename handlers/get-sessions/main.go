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
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{TableName: aws.String(os.Getenv("GAME_SESSION_DYNAMODB"))})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	items := []game.GameSessionDDBItem{}
	attributevalue.UnmarshalListOfMaps(out.Items, &items)
	body, _ := json.Marshal(items)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    game.DefaultCORSHeaders,
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}
