package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sam-app/game"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

func handler(ctx context.Context, req events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	cookie := req.Headers["Cookie"]
	log.Print(cookie)
	if !strings.Contains(cookie, "GreatKingdomAuth") {
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("no auth headers")
	}
	payload, _, _ := strings.Cut(cookie[strings.Index(cookie, "GreatKingdomAuth=")+17:], ";")
	tokenType, token, _ := strings.Cut(payload, " ")
	if strings.Compare(tokenType, "bearer") != 0 {
		return events.APIGatewayCustomAuthorizerResponse{}, fmt.Errorf("token type not matching")
	}
	claim := game.AuthTokenClaims{}
	t, err := jwt.ParseWithClaims(token, &claim, func(t *jwt.Token) (interface{}, error) {
		return []byte("key"), nil
	})
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	}
	if t.Valid {
		return events.APIGatewayCustomAuthorizerResponse{
			PrincipalID: claim.UserId,
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Version: "2012-10-17",
				Statement: []events.IAMPolicyStatement{
					{
						Action:   []string{"execute-api:Invoke"},
						Effect:   "Allow",
						Resource: []string{fmt.Sprintf("%s:%s/*", os.Getenv("ALLOWED_RESOURCES_PREFIX"), req.RequestContext.APIID)},
					},
				},
			},
			Context: map[string]interface{}{"UserId": claim.UserId},
		}, nil
	}
	return events.APIGatewayCustomAuthorizerResponse{}, nil
}

func main() {
	lambda.Start(handler)
}
