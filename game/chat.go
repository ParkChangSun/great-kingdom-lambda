package game

type Chat struct {
	// UserId string
	Timestamp     int64  `json:"timestamp"`
	ConnectionId  string `json:"connectionId"`
	GameSessionId string `json:"gameSessionId"`
	Message       string `json:"message"`
}

func NewChat(connectionId string, gameSessionId string, message string) {

}
