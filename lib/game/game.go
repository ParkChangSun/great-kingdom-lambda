package game

import (
	"slices"
	"time"
)

type Game struct {
	Turn          int
	PassFlag      bool
	Playing       bool
	Board         [9][9]CellStatus
	CoinToss      []string
	LastMoveTime  int64
	RemainingTime []int64
	LastMove      *Point
}

func (g *Game) StartNewGame(players []string) {
	g.Turn = 1
	g.PassFlag = false
	g.Playing = true
	g.Board = [9][9]CellStatus{}
	g.Board[4][4] = NEUTRALCASTLE
	nowMilli := time.Now().UnixMilli()
	g.CoinToss = slices.Clone(players)
	if nowMilli%2 == 0 {
		slices.Reverse(g.CoinToss)
	}
	t, _ := time.ParseDuration("5m")
	g.RemainingTime = []int64{t.Milliseconds(), t.Milliseconds()}
	g.LastMoveTime = nowMilli
	g.LastMove = nil
}
