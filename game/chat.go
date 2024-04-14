package game

import "github.com/golang-jwt/jwt/v5"

type ServerToClient struct {
	EventType string
	Chat      string
	Game
	Players            []User
	CurrentConnections []User
}

type AuthTokenClaims struct {
	jwt.RegisteredClaims
	UserId string
}
