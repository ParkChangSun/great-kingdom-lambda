package auth

import (
	"great-kingdom-lambda/lib/vars"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ACCESSEXPIRES  = time.Minute * 5
	REFRESHEXPIRES = time.Hour
	EXPIRED        = time.Hour * -1
)

type Authenticate struct {
	Id, Password string
}

type AuthBody struct {
	Authorized      bool
	AccessToken, Id string
}

var CORSHeaders = map[string]string{
	"Access-Control-Allow-Credentials": "true",
	"Access-Control-Allow-Origin":      vars.CLIENT_ORIGIN,
}

var ExpiredCookie = CookieHeader("GreatKingdomRefresh", "", time.Now().Add(EXPIRED))

func AuthHeaders(r string) map[string]string {
	return map[string]string{
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Origin":      vars.CLIENT_ORIGIN,
		"Set-Cookie":                       CookieHeader("GreatKingdomRefresh", r, time.Now().Add(REFRESHEXPIRES)),
	}
}

func CookieHeader(name string, value string, expires time.Time) string {
	authCookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   "greatkingdom.net",
		Path:     "/",
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
	}
	return authCookie.String()
}

func ParseToken(k string) (*jwt.Token, jwt.RegisteredClaims, error) {
	c := jwt.RegisteredClaims{}
	t, err := jwt.ParseWithClaims(k, &c, func(t *jwt.Token) (interface{}, error) { return []byte(vars.JWT_SIGNING_KEY), nil })
	return t, c, err
}

func GenerateTokenSet(userId string) (string, string, error) {
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userId,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ACCESSEXPIRES)),
	})
	signedAccess, err := access.SignedString([]byte(vars.JWT_SIGNING_KEY))
	if err != nil {
		return "", "", err
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userId,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(REFRESHEXPIRES)),
	})
	signedRefresh, err := refresh.SignedString([]byte(vars.JWT_SIGNING_KEY))
	if err != nil {
		return "", "", err
	}
	return signedAccess, signedRefresh, nil
}

func RESTResponse(statusCode int, headers map[string]string, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}

type ErrorResponseBody struct {
	Message string `json:"message"`
}
