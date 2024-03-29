package main

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(r events.APIGatewayWebsocketProxyRequest) {
	log.Print("connected!")
}

func main() {
	lambda.Start(handler)
}
