package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayCustomAuthorizerRequestTypeRequest) {
	token := req.Headers["Cookie"]
	fmt.Print(token)
}

func main() {
	lambda.Start(handler)
}
