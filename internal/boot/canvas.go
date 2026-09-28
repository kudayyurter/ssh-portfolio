package boot

import "strings"

// tone is a cell's color. plain uses the terminal's default foreground.
type tone uint8

const (
	plain tone = iota
	faint      // #808080
	muted      // #a3a3a3
)

var sgr = map[tone]string{
	faint: "\x1b[38;2;128;128;128m",
	muted: "\x1b[38;2;163;163;163m",
}

// canvas is a w×h grid of styled cells.
type canvas struct {
	w, h  int
	cells [][]rune
	tones [][]tone
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, cells: make([][]rune, h), tones: make([][]tone, h)}
	for y := range h {
		c.cells[y] = []rune(strings.Repeat(" ", w))
		c.tones[y] = make([]tone, w)
	}
	return c
}

func (c *canvas) set(x, y int, r rune, t tone) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.cells[y][x], c.tones[y][x] = r, t
	}
}

func (c *canvas) empty(x, y int) bool {
	return x >= 0 && x < c.w && y >= 0 && y < c.h && c.cells[y][x] == ' '
}

// text writes s starting at x (each rune is one column).
func (c *canvas) text(x, y int, s string, t tone) {
	for i, r := range []rune(s) {
		c.set(x+i, y, r, t)
	}
}

// String renders rows joined by newlines, emitting a color code only where
// the tone changes so frames stay small.
func (c *canvas) String() string {
	var b strings.Builder
	for y := range c.h {
		if y > 0 {
			b.WriteByte('\n')
		}
		cur := plain
		for x := range c.w {
			if t := c.tones[y][x]; t != cur {
				if cur != plain {
					b.WriteString("\x1b[m")
				}
				if t != plain {
					b.WriteString(sgr[t])
				}
				cur = t
			}
			b.WriteRune(c.cells[y][x])
		}
		if cur != plain {
			b.WriteString("\x1b[m")
		}
	}
	return b.String()
}
