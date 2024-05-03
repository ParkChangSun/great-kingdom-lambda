package game

import "github.com/golang-jwt/jwt/v5"

type AuthTokenClaims struct {
	jwt.RegisteredClaims
	UserId string
}
