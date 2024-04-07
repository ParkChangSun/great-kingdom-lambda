package game

type QueueEvent int

type ConnectionDDBItem struct {
	ConnectionId  string
	Timestamp     int64
	GameSessionId string
	UserId        string
}

type Move struct {
	Point
	Pass bool
}

type GameMoveSQSRecord struct {
	Timestamp     int64
	ConnectionId  string
	GameSessionId string
	Move          Move
}
