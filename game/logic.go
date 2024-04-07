package game

import (
	"fmt"
	"math/rand"
	"time"
)

var neighbors = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

type Board [9][9]CellStatus

func (g Game) getPlayers() (first, last CellStatus) {
	if g.turn%2 == 1 {
		return FirstCastle, LastCastle
	} else {
		return LastCastle, FirstCastle
	}
}

func (b *Board) putPiece(x, y int, s CellStatus) {
	b[y][x] = s
}

func (g *Game) checkSieged(x, y int, siegedFlag CellStatus) bool {
	if _, b := g.checkedList[[2]int{x, y}]; b {
		return true
	}

	uncheckedList := [][2]int{}

	for _, v := range neighbors {
		dx, dy := x+v[0], y+v[1]
		c, _ := g.board.getCoordinate(dx, dy)
		if c == 0 {
			return false
		} else if c == siegedFlag {
			uncheckedList = append(uncheckedList, [2]int{dx, dy})
		}
	}

	g.checkedList[[2]int{x, y}] = true

	for _, v := range uncheckedList {
		if !g.checkSieged(v[0], v[1], siegedFlag) {
			return false
		}
	}
	return true
}

func (g Game) checkOccupation(x, y int) bool {
	if _, b := g.checkedList[[2]int{x, y}]; b {
		return true
	}

	uncheckedList := [][2]int{}

	for _, v := range neighbors {
		dx, dy := x+v[0], y+v[1]
		c, e := g.board.getCoordinate(dx, dy)
		if _, o := g.getPlayers(); c == o {
			return false
		} else if c == 0 {
			uncheckedList = append(uncheckedList, [2]int{dx, dy})
		} else if e != 0 {
			g.edgeCount[e] = true

			if g.edgeCount[1] && g.edgeCount[2] && g.edgeCount[3] && g.edgeCount[4] {
				return false
			}
		}
	}

	g.checkedList[[2]int{x, y}] = true

	for _, v := range uncheckedList {
		if !g.checkOccupation(v[0], v[1]) {
			return false
		}
	}

	return true
}

func (b Board) getCoordinate(x, y int) (cellStatus CellStatus, boardEdge int) {
	if x > 8 {
		return 3, 1
	} else if x < 0 {
		return 3, 3
	} else if y > 8 {
		return 3, 2
	} else if y < 0 {
		return 3, 4
	} else {
		return b[y][x], 0
	}
}

func (b Board) debug() {
	fmt.Println("yx 0 1 2 3 4 5 6 7 8")
	for i, v := range b {
		fmt.Println(i, v)
	}
	fmt.Print("\n---------------------\n")
}

func (g *Game) SStart() {
	g.board.putPiece(4, 4, Neutral)
	g.turn = 1

	// coin toss to set first player

}

func (g *Game) Start(host, guest string) {
	g.board.putPiece(4, 4, Neutral)
	g.turn = 1

	if rand.New(rand.NewSource(time.Now().UnixNano())).Intn(2)%2 == 1 {
		g.players = [2]string{host, guest}
	} else {
		g.players = [2]string{guest, host}
	}
	fmt.Printf("coin toss %s\n", g.players)

	for g.winner == 0 {
		player, opponent := g.getPlayers()
		g.board.debug()

		fmt.Printf("Turn: %d\nPlayer: %s\n", g.turn, g.players[int(player)-1])

		// 모든 수는 유효하다
		// 따라서 보드를 abort하는 경우 없음
		var x, y int
		fmt.Scanln(&x, &y)
		g.board.putPiece(x, y, player)

		for _, v := range neighbors {
			dx, dy := x+v[0], y+v[1]
			g.checkedList = map[[2]int]bool{}
			g.edgeCount = map[int]bool{}

			if c, _ := g.board.getCoordinate(dx, dy); c == opponent {
				if g.checkSieged(dx, dy, opponent) {
					g.winner = int(player)
					break
				}
			} else if c == EmptyCell {
				if g.checkOccupation(dx, dy) {
					for i := range g.checkedList {
						g.board.putPiece(i[0], i[1], CellStatus(player+3))
					}
				}
			}
		}

		g.turn++
	}

	g.board.debug()
	fmt.Println("winner", g.winner)
}
