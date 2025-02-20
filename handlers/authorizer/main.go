package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func generatePolicy(principalID, effect, resource string, context map[string]interface{}) events.APIGatewayCustomAuthorizerResponse {
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
	cookie := req.Headers["Cookie"]

	if !strings.Contains(cookie, "GreatKingdomAuth=") {
		return events.APIGatewayCustomAuthorizerResponse{}, nil
	}

	accessTokenStr, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomAuth=")+17:], ";")
	accessTokenClaims := jwt.RegisteredClaims{}
	accessToken, err := jwt.ParseWithClaims(accessTokenStr, &accessTokenClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte("key"), nil
	})
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	}

	resource := fmt.Sprintf("%s:%s/%s/*", os.Getenv("ALLOWED_RESOURCES_PREFIX"), req.RequestContext.APIID, req.RequestContext.Stage)

	if accessToken.Valid {
		return generatePolicy(accessTokenClaims.Subject, "Allow", resource, map[string]interface{}{"UserId": accessTokenClaims.Subject}), nil
	}
	return generatePolicy(accessTokenClaims.Subject, "Deny", resource, nil), nil
}

func main() {
	lambda.Start(handler)
}
