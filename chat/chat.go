package chat

type Msg struct {
	Timestamp int64 `json:"timestamp"`
	// UserId string?
	Id     string `json:"id"`
	RoomId string `json:"roomId"`
	Chat   string `json:"chat"`
}
