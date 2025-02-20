package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var CORSHeaders = map[string]string{
	"Access-Control-Allow-Credentials": "true",
	"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
}

func AuthHeaders(a, r string) map[string]string {
	return map[string]string{
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
		"Set-Cookie":                       CookieHeader("GreatKingdomRefresh", r, time.Now().Add(REFRESHEXPIRES)),
		"Authorization":                    a,
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

const (
	ACCESSEXPIRES  = time.Minute * 5
	REFRESHEXPIRES = time.Minute * 15
	EXPIRED        = time.Hour * -1
)

func GenerateTokenSet(userId string) (string, string, error) {
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userId,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ACCESSEXPIRES)),
	})
	signedAccess, err := access.SignedString([]byte("key"))
	if err != nil {
		return "", "", err
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userId,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ACCESSEXPIRES)),
	})
	signedRefresh, err := refresh.SignedString([]byte("key"))
	if err != nil {
		return "", "", err
	}
	return signedAccess, signedRefresh, nil
}
