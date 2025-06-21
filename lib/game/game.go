package game

import "fmt"

type Point struct {
	R, C int
}

type Move struct {
	Point Point
	Pass  bool
}

// turn도 레코드 길이로 할수있는데?
type GameTable struct {
	Turn   int
	Board  [9][9]CellStatus
	Record []Move
	Result string
}

func NewGameTable() *GameTable {
	g := &GameTable{
		Turn:   1,
		Record: []Move{},
	}
	g.Board[4][4] = NEUTRALCASTLE
	return g
}

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

func (p Point) getNeighbors() []Point {
	return []Point{{p.R, p.C - 1}, {p.R, p.C + 1}, {p.R - 1, p.C}, {p.R + 1, p.C}}
}

func (g *GameTable) putPiece(p Point, s CellStatus) {
	g.Board[p.R][p.C] = s
}

func (g GameTable) getCellStatus(p Point) (cell CellStatus, edge int) {
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

func (g GameTable) getPlayerColor() (attacker CellStatus, defenser CellStatus) {
	if g.Turn%2 == 1 {
		return BLUECASTLE, ORANGECASTLE
	} else {
		return ORANGECASTLE, BLUECASTLE
	}
}

// given point is side of moved point
func (g GameTable) checkSieged(defenserCell Point) map[Point]struct{} {
	attackerColor, defenserColor := g.getPlayerColor()
	if c, _ := g.getCellStatus(defenserCell); c != defenserColor {
		return nil
	}

	checked := map[Point]struct{}{}
	queue := []Point{defenserCell}
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

func (g GameTable) checkOccupied(defenserCell Point) map[Point]struct{} {
	attackerColor, _ := g.getPlayerColor()
	if c, _ := g.getCellStatus(defenserCell); c != EMPTYLAND {
		return nil
	}

	checked := map[Point]struct{}{}
	queue := []Point{defenserCell}
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
	c, _ := g.getCellStatus(m.Point)
	return c == EMPTYLAND
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

	if p.Pass {
		if len(g.Record) >= 2 && g.Record[len(g.Record)-2].Pass {
			blue, orange := g.Count()
			g.Result = fmt.Sprint(blue, ":", orange, "(2.5)")
			if blue > orange+2 {
				return 0
			} else {
				return 1
			}
		} else {
			g.Turn++
			return -1
		}
	}

	attackerColor, _ := g.getPlayerColor()
	g.putPiece(p.Point, attackerColor)

	sieged := false
	for _, n := range p.Point.getNeighbors() {
		if s := g.checkSieged(n); s != nil {
			for c := range s {
				g.putPiece(c, SIEGED)
				sieged = true
			}
		}
		if o := g.checkOccupied(n); o != nil {
			for c := range o {
				g.putPiece(c, attackerColor+TERRITORYOFFSET)
			}
		}
	}

	// if sieged game over
	if sieged {
		if (g.Turn-1)%2 == 0 {
			g.Result = "blue besieged opponent"
			return 0
		} else {
			g.Result = "orange besieged opponent"
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
		g.Result = fmt.Sprint(blue, ":", orange, "(+2.5)")
		if blue > orange+2 {
			return 0
		} else {
			return 1
		}
	}

	g.Turn++
	return -1
}
