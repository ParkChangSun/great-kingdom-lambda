package game

import (
	"fmt"
	"log"
)

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

type Game struct {
	Turn      int
	PassFlag  bool
	Playing   bool
	Board     [9][9]CellStatus
	PlayersId [2]string
}

type Point struct {
	R int
	C int
}

type Move struct {
	Point Point
	Pass  bool
	Start bool
}

func (p Point) getNeighbors() []Point {
	return []Point{{p.R, p.C - 1}, {p.R, p.C + 1}, {p.R - 1, p.C}, {p.R + 1, p.C}}
}

func (g *Game) putPiece(p Point, s CellStatus) {
	g.Board[p.R][p.C] = s
}

func (g Game) getCellStatus(p Point) (cell CellStatus, edge int) {
	if p.R < 0 {
		return Edge, 0
	}
	if p.R > 8 {
		return Edge, 1
	}
	if p.C < 0 {
		return Edge, 2
	}
	if p.C > 8 {
		return Edge, 3
	}
	return g.Board[p.R][p.C], -1
}

func (g Game) getPlayerColor() (attacker CellStatus, defenser CellStatus) {
	if g.Turn%2 == 1 {
		return BlueCastle, OrangeCastle
	} else {
		return OrangeCastle, BlueCastle
	}
}

// given point is side of moved point
func (g Game) checkSieged(p Point) map[Point]struct{} {
	_, defenser := g.getPlayerColor()
	if c, _ := g.getCellStatus(p); c != defenser {
		return nil
	}

	checkedList := map[Point]struct{}{}
	checkQueue := []Point{p}
	edgeCheck := [4]bool{}

	for len(checkQueue) > 0 {
		item := checkQueue[0]
		checkQueue = checkQueue[1:]

		if _, b := checkedList[item]; b {
			continue
		}
		checkedList[item] = struct{}{}

		for _, n := range item.getNeighbors() {
			stat, edge := g.getCellStatus(n)
			if stat == EmptyCell {
				return nil
			}
			if stat == defenser {
				checkQueue = append(checkQueue, n)
			}
			if stat == Edge {
				edgeCheck[edge] = true
				if edgeCheck[0] && edgeCheck[1] && edgeCheck[2] && edgeCheck[3] {
					return nil
				}
			}
		}
	}

	return checkedList
}

func (g Game) checkOccupied(p Point) map[Point]struct{} {
	if c, _ := g.getCellStatus(p); c != EmptyCell {
		return nil
	}

	_, defenser := g.getPlayerColor()
	checkedList := map[Point]struct{}{}
	checkQueue := []Point{p}
	edgeCheck := [4]bool{}

	for len(checkQueue) > 0 {
		item := checkQueue[0]
		checkQueue = checkQueue[1:]

		if _, b := checkedList[item]; b {
			continue
		}
		checkedList[item] = struct{}{}

		for _, n := range item.getNeighbors() {
			stat, edge := g.getCellStatus(n)
			if stat == EmptyCell {
				checkQueue = append(checkQueue, n)
			}
			if stat == defenser {
				log.Print("point ", p, " return nil faced enemy when checklist ", checkedList)
				return nil
			}
			if stat == Edge {
				edgeCheck[edge] = true
				if edgeCheck[0] && edgeCheck[1] && edgeCheck[2] && edgeCheck[3] {
					log.Print("point ", p, " return nil edge check when checklist ", checkedList)
					return nil
				}
			}
		}
	}

	return checkedList
}

func (g *Game) Move(p Point) (finished bool, err error) {
	if c, _ := g.getCellStatus(p); c != EmptyCell {
		return false, fmt.Errorf(fmt.Sprintf("%+v is not playable point", p))
	}

	g.PassFlag = false
	attacker, _ := g.getPlayerColor()
	g.putPiece(p, attacker)

	for _, n := range p.getNeighbors() {
		if s := g.checkSieged(n); s != nil {
			finished = true
			for t := range s {
				g.putPiece(t, SIEGED)
			}
		}
	}
	if finished {
		g.Playing = false
		return
	}

	for _, n := range p.getNeighbors() {
		if o := g.checkOccupied(n); o != nil {
			for c := range o {
				g.putPiece(c, attacker+CELLSTATUSOFFSET)
			}
		}
	}

	g.Turn++

	return
}

func (g *Game) Pass() (finished bool) {
	if g.PassFlag {
		g.Playing = false
		return true
	}
	g.PassFlag = true
	g.Turn++
	return false
}

func (g Game) CountTerritory() (blue int, orange int) {
	for _, r := range g.Board {
		for _, c := range r {
			if c == BlueTerritory {
				blue++
			}
			if c == OrangeTerritory {
				orange++
			}
		}
	}
	return
}
