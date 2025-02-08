package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AUTHEXPIRES    = time.Minute * 5
	REFRESHEXPIRES = time.Minute * 15
	EXPIRED        = time.Hour * -1
)

type AuthTokenClaims struct {
	jwt.RegisteredClaims
	UserId string
}

func NewAuthToken(userId string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthTokenClaims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(AUTHEXPIRES),
			},
		},
	})
	signedToken, err := token.SignedString([]byte("key"))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

type RefreshToken struct {
	RefreshId string
	Time      time.Time
}

func NewRefreshToken() string {
	t, _ := json.Marshal(RefreshToken{
		RefreshId: uuid.NewString(),
		Time:      time.Now(),
	})
	return base64.RawStdEncoding.EncodeToString(t)
}

func GetCookieHeader(name string, value string, expires time.Time) string {
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

func ParseRefreshToken(cookieString string) string {
	if !strings.Contains(cookieString, "GreatKingdomRefresh=") {
		return ""
	}
	payload, _, _ := strings.Cut(cookieString[strings.Index(cookieString, "GreatKingdomRefresh=")+20:], ";")
	return payload
}

var DefaultCORSHeaders = map[string]string{
	"Access-Control-Allow-Credentials": "true",
	"Access-Control-Allow-Origin":      os.Getenv("WEB_CLIENT_ORIGIN"),
}

var SignOutResponse = events.APIGatewayProxyResponse{
	StatusCode: 401,
	Headers:    DefaultCORSHeaders,
	MultiValueHeaders: map[string][]string{
		"Set-Cookie": {
			GetCookieHeader("GreatKingdomAuth", "logout", time.Now().Add(EXPIRED)),
			GetCookieHeader("GreatKingdomRefresh", "logout", time.Now().Add(EXPIRED)),
		},
	},
}
