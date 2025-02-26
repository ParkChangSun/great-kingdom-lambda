package main

import (
	"context"

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
	// resource := fmt.Sprintf("%s:%s/%s/*", os.Getenv("ALLOWED_RESOURCES_PREFIX"), req.RequestContext.APIID, req.RequestContext.Stage)

	authorization := req.Headers["Authorization"]
	accessTokenClaims := jwt.RegisteredClaims{}
	accessToken, err := jwt.ParseWithClaims(authorization, &accessTokenClaims, func(t *jwt.Token) (any, error) {
		return []byte("key"), nil
	})
	if err != nil {
		return generatePolicy(accessTokenClaims.Subject, "Deny", req.MethodArn, nil), nil
	}

	if accessToken.Valid {
		return generatePolicy(accessTokenClaims.Subject, "Allow", req.MethodArn, map[string]any{"UserId": accessTokenClaims.Subject}), nil
	}
	return generatePolicy(accessTokenClaims.Subject, "Deny", req.MethodArn, nil), nil
}

func main() {
	lambda.Start(handler)
}
