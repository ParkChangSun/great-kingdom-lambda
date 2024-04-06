package game

type QueueEvent int

type JoinRecord struct {
	ConnectionId  string
	Timestamp     int64
	GameSessionId string
	UserId        string
}
