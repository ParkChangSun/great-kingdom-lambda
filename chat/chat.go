package chat

type Chat struct {
	Timestamp int64 `json:"timestamp"`
	// UserId string?
	ConnectionId  string `json:"connectionId"`
	GameSessionId string `json:"gameSessionId"`
	Message       string `json:"message"`
}
