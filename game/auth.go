package game

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

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
	UserId    string
	RefreshId string
	Time      time.Time
}

func NewRefreshToken(userId string) string {
	t, _ := json.Marshal(RefreshToken{
		UserId:    userId,
		RefreshId: uuid.NewString(),
		Time:      time.Now(),
	})
	return base64.RawStdEncoding.EncodeToString(t)
}

func GetCookieHeader(name string, value string, expires time.Time) string {
	authCookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
	}
	return authCookie.String()
}
