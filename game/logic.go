package game

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

type Game struct {
	Turn     int
	PassFlag bool
	Playing  bool
	Board    [9][9]CellStatus
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

func (g Game) getPlayerColor() (attacker CellStatus, defenser CellStatus) {
	if g.Turn%2 == 1 {
		return BLUECASTLE, ORANGECASTLE
	} else {
		return ORANGECASTLE, BLUECASTLE
	}
}

// given point is side of moved point
func (g Game) checkSieged(defenserCell Point) map[Point]struct{} {
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

func (g Game) checkOccupied(defenserCell Point) map[Point]struct{} {
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

func (g Game) CellPlayable(p Point) bool {
	c, _ := g.getCellStatus(p)
	return c == EMPTYLAND
}

func (g *Game) Move(p Point) {
	g.PassFlag = false
	attackerColor, _ := g.getPlayerColor()

	g.putPiece(p, attackerColor)

	sieged := false
	for _, n := range p.getNeighbors() {
		if s := g.checkSieged(n); s != nil {
			sieged = true
			for c := range s {
				g.putPiece(c, SIEGED)
			}
		}
		if o := g.checkOccupied(n); o != nil {
			for c := range o {
				g.putPiece(c, attackerColor+TERRITORYOFFSET)
			}
		}
	}

	if sieged {
		g.Playing = false
		return
	}

	playable := false
	for _, r := range g.Board {
		for _, c := range r {
			if c == EMPTYLAND {
				playable = true
			}
		}
	}
	if !playable {
		g.Playing = false
		return
	}

	g.Turn++
}

func (g *Game) Pass() {
	if g.PassFlag {
		g.Playing = false
		return
	}
	g.PassFlag = true
	g.Turn++
}

func (g Game) CountTerritory() (blue int, orange int, sieged bool, winner int) {
	for _, r := range g.Board {
		for _, c := range r {
			if c == BLUETERRITORY {
				blue++
			}
			if c == ORANGETERRITORY {
				orange++
			}
			if c == SIEGED {
				sieged = true
			}
		}
	}
	if sieged {
		winner = (g.Turn - 1) % 2
	} else if blue >= orange+3 {
		winner = 0
	} else {
		winner = 1
	}
	return
}

func (g *Game) StartNewGame() {
	g.Turn = 1
	g.PassFlag = false
	g.Playing = true
	g.Board = [9][9]CellStatus{}
	g.Board[4][4] = NEUTRALCASTLE
}
