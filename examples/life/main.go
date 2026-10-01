// Command life runs Conway's Game of Life on a fixed toroidal board.
package main

const size = 8

type Board [size][size]bool

func (b *Board) neighbors(r, c int) int {
	n := 0
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			if b[(r+dr+size)%size][(c+dc+size)%size] {
				n++
			}
		}
	}
	return n
}

func (b Board) Step() Board {
	var next Board
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			n := b.neighbors(r, c)
			next[r][c] = n == 3 || b[r][c] && n == 2
		}
	}
	return next
}

func (b Board) Print() {
	for _, row := range b {
		line := ""
		for _, alive := range row {
			if alive {
				line += "#"
			} else {
				line += "."
			}
		}
		println(line)
	}
}

func main() {
	var b Board
	for _, p := range [][2]int{{0, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}} {
		b[p[0]][p[1]] = true
	}
	start := b
	for gen := 0; gen < 4; gen++ {
		b = b.Step()
	}
	b.Print()
	println("changed:", b != start)
}
