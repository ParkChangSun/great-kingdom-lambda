package awsutils

import "sam-app/game"

type GameMoveSQSRecord struct {
	Timestamp     int64
	ConnectionId  string
	UserId        string
	GameSessionId string
	Move          game.Move
}

type GameChatSQSRecord struct {
	Timestamp     int64
	ConnectionId  string
	UserId        string
	GameSessionId string
	Chat          string
}

const (
	JOINEVENT  = "JOIN"
	LEAVEEVENT = "LEAVE"
	CHATEVENT  = "CHAT"
	GAMEEVENT  = "GAME"
	USEREVENT  = "USER"
	SLOTEVENT  = "SLOT"

	GLOBALCHAT = "GLOBAL"
)
