package game

type WebSocketClient struct {
	ConnectionId  string
	GameSessionId string
	UserId        string
}

type CellStatus int

const (
	EmptyCell CellStatus = iota
	Neutral
	BlueCastle
	OrangeCastle
	BlueTerritory
	OrangeTerritory
	SIEGED
	Edge
)

const CELLSTATUSOFFSET = 2

type Point struct {
	R int `json:"r"`
	C int `json:"c"`
}
