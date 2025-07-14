package game

import "fmt"

type CellStatus int

const (
	EMPTYLAND CellStatus = iota
	NEUTRALCASTLE
	BLUECASTLE
	ORANGECASTLE
	BLUETERRITORY
	ORANGETERRITORY
	SIEGED
	EDGE
)

const TERRITORYOFFSET = 2

type Cell struct {
	R, C int
}

func (p Cell) getNeighbors() []Cell {
	return []Cell{{p.R, p.C - 1}, {p.R, p.C + 1}, {p.R - 1, p.C}, {p.R + 1, p.C}}
}

type Move struct {
	Cell Cell
	Pass bool
}

type GameTable struct {
	Board  [9][9]CellStatus
	Record []Move
	Result string
}

func NewGameTable() *GameTable {
	g := &GameTable{
		Record: []Move{},
	}
	g.Board[4][4] = NEUTRALCASTLE
	return g
}

func (g *GameTable) putPiece(p Cell, s CellStatus) {
	g.Board[p.R][p.C] = s
}

func (g GameTable) getCellStatus(p Cell) (cell CellStatus, edge int) {
	if p.R < 0 {
		return EDGE, 0
	}
	if p.R > 8 {
		return EDGE, 1
	}
	if p.C < 0 {
		return EDGE, 2
	}
	if p.C > 8 {
		return EDGE, 3
	}
	return g.Board[p.R][p.C], -1
}

// after record append
func (g GameTable) getPlayerColor() (attacker CellStatus, defenser CellStatus) {
	if len(g.Record)%2 == 0 {
		return ORANGECASTLE, BLUECASTLE
	} else {
		return BLUECASTLE, ORANGECASTLE
	}
}

// given point is side of moved point
func (g GameTable) checkSideSieged(side Cell) map[Cell]struct{} {
	attackerColor, defenserColor := g.getPlayerColor()
	if c, _ := g.getCellStatus(side); c != defenserColor {
		return nil
	}

	checked := map[Cell]struct{}{}
	queue := []Cell{side}
	edgeCheck := [4]bool{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if _, b := checked[item]; b {
			continue
		}
		checked[item] = struct{}{}

		for _, n := range item.getNeighbors() {
			stat, edge := g.getCellStatus(n)
			if stat == defenserColor {
				queue = append(queue, n)
			} else if stat == EDGE {
				edgeCheck[edge] = true
				if edgeCheck[0] && edgeCheck[1] && edgeCheck[2] && edgeCheck[3] {
					return nil
				}
			} else if stat == attackerColor || stat == NEUTRALCASTLE {
				continue
			} else {
				return nil
			}
		}
	}

	return checked
}

func (g GameTable) checkSideOccupied(side Cell) map[Cell]struct{} {
	attackerColor, _ := g.getPlayerColor()
	if c, _ := g.getCellStatus(side); c != EMPTYLAND {
		return nil
	}

	checked := map[Cell]struct{}{}
	queue := []Cell{side}
	edgeCheck := [4]bool{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if _, b := checked[item]; b {
			continue
		}
		checked[item] = struct{}{}

		for _, n := range item.getNeighbors() {
			stat, edge := g.getCellStatus(n)
			if stat == EMPTYLAND {
				queue = append(queue, n)
			} else if stat == EDGE {
				edgeCheck[edge] = true
				if edgeCheck[0] && edgeCheck[1] && edgeCheck[2] && edgeCheck[3] {
					return nil
				}
			} else if stat == attackerColor || stat == NEUTRALCASTLE {
				continue
			} else {
				return nil
			}
		}
	}

	return checked
}

func (g GameTable) Playable(m Move) bool {
	if m.Pass {
		return true
	}
	c, _ := g.getCellStatus(m.Cell)
	if c != EMPTYLAND {
		return false
	}

	var attacker CellStatus
	if len(g.Record)%2 == 0 {
		attacker, _ = BLUECASTLE, ORANGECASTLE
	} else {
		attacker, _ = ORANGECASTLE, BLUECASTLE
	}

	g.putPiece(m.Cell, attacker)
	g.Record = append(g.Record, m)

	for _, n := range m.Cell.getNeighbors() {
		if s := g.checkSideSieged(n); s != nil {
			return true
		}
	}

	checked := map[Cell]struct{}{}
	queue := []Cell{m.Cell}

	for len(queue) != 0 {
		cur := queue[0]
		queue = queue[1:]

		if _, ok := checked[cur]; ok {
			continue
		}
		checked[cur] = struct{}{}

		for _, v := range cur.getNeighbors() {
			c, _ := g.getCellStatus(v)
			if c == EMPTYLAND {
				return true
			} else if c == attacker {
				queue = append(queue, v)
			}
		}
	}

	return false
}

func (g GameTable) Count() (int, int) {
	blue, orange := 0, 0
	for _, r := range g.Board {
		for _, c := range r {
			if c == BLUETERRITORY {
				blue++
			}
			if c == ORANGETERRITORY {
				orange++
			}
		}
	}
	return blue, orange
}

func (g *GameTable) MakeMove(p Move) int {
	g.Record = append(g.Record, p)
	attackerColor, _ := g.getPlayerColor()

	if p.Pass {
		if len(g.Record) >= 2 && g.Record[len(g.Record)-2].Pass {
			blue, orange := g.Count()
			g.Result = fmt.Sprint(blue, ":", orange+2, ".5")
			if blue > orange+2 {
				return 0
			} else {
				return 1
			}
		} else {
			return -1
		}
	}

	g.putPiece(p.Cell, attackerColor)

	sieged := false
	for _, n := range p.Cell.getNeighbors() {
		if s := g.checkSideSieged(n); s != nil {
			for c := range s {
				g.putPiece(c, SIEGED)
				sieged = true
			}
		}
		if o := g.checkSideOccupied(n); o != nil {
			for c := range o {
				g.putPiece(c, attackerColor+TERRITORYOFFSET)
			}
		}
	}

	// if sieged game over
	if sieged {
		g.Result = "SIEGED"
		if len(g.Record)%2 == 1 {
			return 0
		} else {
			return 1
		}
	}

	// if board is full game over
	var empty int
	for _, r := range g.Board {
		for _, c := range r {
			if c == EMPTYLAND {
				empty++
			}
		}
	}
	if empty == 0 {
		blue, orange := g.Count()
		if blue > orange+2 {
			g.Result = fmt.Sprint(blue, ":", orange+2, ".5")
			return 0
		} else {
			g.Result = fmt.Sprint(blue, ":", orange+2, ".5")
			return 1
		}
	}

	return -1
}
