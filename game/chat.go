package game

type ServerToClient struct {
	EventType string
	Chat      string
	Game
	Players            []User
	CurrentConnections []User
}
