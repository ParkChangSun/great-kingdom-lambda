package main

import (
	"context"
	"fmt"
	"great-kingdom-lambda/lib/auth"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func generatePolicy(principalID, effect, resource string, context map[string]any) events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
		Context: context,
	}
}

func handler(ctx context.Context, req events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	resource := fmt.Sprintf("arn:aws:execute-api:*:%s:%s/*", req.RequestContext.AccountID, req.RequestContext.APIID)

	_, claims, err := auth.ParseToken(req.Headers["authorization"])
	if err != nil {
		log.Print(req.Headers)
		log.Print(err)
		return generatePolicy(claims.Subject, "Deny", resource, nil), nil
	}

	return generatePolicy(claims.Subject, "Allow", resource, map[string]any{"UserId": claims.Subject}), nil
}

func main() {
	lambda.Start(handler)
}
