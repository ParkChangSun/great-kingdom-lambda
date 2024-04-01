package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var greeting string
	sourceIP := request.RequestContext.Identity.SourceIP

	if sourceIP == "" {
		greeting = "Hello, world!\n"
	} else {
		greeting = fmt.Sprintf("Hello, %s!\n", sourceIP)
	}

	return events.APIGatewayProxyResponse{
		Body:       greeting,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

// var wsClient *apigatewaymanagementapi.Client
// wsClient = apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
// 	o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s/%s", req.RequestContext.DomainName, req.RequestContext.Stage))
// })
// _, err = wsClient.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
// 	ConnectionId: aws.String(req.RequestContext.ConnectionID),
// 	Data:         []byte(fmt.Sprintf("%s connected", req.RequestContext.ConnectionID)),
// })
// if err != nil {
// 	return events.APIGatewayProxyResponse{}, err
// }
