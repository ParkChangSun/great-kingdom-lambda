package main

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Print("connected!", r.RequestContext.Identity.SourceIP)
	return events.APIGatewayProxyResponse{Body: r.RequestContext.Identity.SourceIP, StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
