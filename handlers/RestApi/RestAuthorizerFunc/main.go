package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
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

	accessTokenClaims := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(req.Headers["Authorization"], &accessTokenClaims, func(t *jwt.Token) (any, error) {
		return []byte("key"), nil
	})
	if err != nil {
		return generatePolicy(accessTokenClaims.Subject, "Deny", resource, nil), nil
	}

	return generatePolicy(accessTokenClaims.Subject, "Allow", resource, map[string]any{"UserId": accessTokenClaims.Subject}), nil
}

func main() {
	lambda.Start(handler)
}
