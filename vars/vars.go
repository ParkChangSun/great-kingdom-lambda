package vars

import "os"

type TABLEEVENTTYPE int

const (
	TABLEJOINEVENT TABLEEVENTTYPE = iota
	TABLELEAVEEVENT
	TABLECHATEVENT
	TABLEMOVEEVENT
	TABLESLOTEVENT
)

const (
	TABLEBROADCAST = "TABLE"
	CHATBROADCAST  = "CHAT"
	PONGBROADCAST  = "PONG"
	AUTHBROADCAST  = "AUTH"
)

var (
	GAME_TABLE_DYNAMODB    = os.Getenv("GAME_TABLE_DYNAMODB")
	USER_DYNAMODB          = os.Getenv("USER_DYNAMODB")
	CONNECTION_DYNAMODB    = os.Getenv("CONNECTION_DYNAMODB")
	GAME_TABLE_EVENT_QUEUE = os.Getenv("GAME_TABLE_EVENT_QUEUE")
	CLIENT_ORIGIN          = os.Getenv("CLIENT_ORIGIN")
	WEBSOCKET_ENDPOINT     = os.Getenv("WEBSOCKET_ENDPOINT")
)
