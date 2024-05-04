package game

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
				Time: time.Now().Add(time.Minute * 15),
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
