package game

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	JOINEVENT  = "JOIN"
	LEAVEEVENT = "LEAVE"
	CHATEVENT  = "CHAT"
)

type User struct {
	ConnectionId string
	UserId       string
}

type GameSession struct {
	GameSessionId      string
	GameSessionName    string
	CurrentConnections []User
	Players            []User
	// Game
}

func GetGameSessionDynamoDBKey(gameSessionId string) map[string]types.AttributeValue {
	key, _ := attributevalue.MarshalMap(struct{ GameSessionId string }{gameSessionId})
	return key
}

func GetConnectionDynamoDBKey(connectionId string) map[string]types.AttributeValue {
	key, _ := attributevalue.MarshalMap(struct{ ConnectionId string }{connectionId})
	return key
}

type Game struct {
	Board    [9][9]CellStatus
	Turn     int
	PassFlag bool
	Playing  bool
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
