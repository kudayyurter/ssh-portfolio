// Package pixelfont holds the 5×7 pixel glyphs shared with kudayyurter.dev
// (Portfolio/src/lib/pixel-font.ts) and turns text into 1-bit bitmaps.
package pixelfont

import (
	"fmt"
	"strings"
)

// Bitmap is a 1-bit image: one string per row, '1' = filled pixel.
type Bitmap []string

// Width is the number of pixel columns.
func (b Bitmap) Width() int {
	if len(b) == 0 {
		return 0
	}
	return len(b[0])
}

// Height of every glyph.
const Height = 7

// Glyphs are copied verbatim from the website so both render the same name.
var Glyphs = map[rune]Bitmap{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "01010", "01010", "00100"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'-': {"00000", "00000", "00000", "01110", "00000", "00000", "00000"},
}

// Text lays glyphs out left to right with a one-column gap.
func Text(s string) (Bitmap, error) {
	rows := make(Bitmap, Height)
	for i, r := range []rune(s) {
		g, ok := Glyphs[r]
		if !ok {
			return nil, fmt.Errorf("pixelfont: no glyph for %q", r)
		}
		for y := range Height {
			if i > 0 {
				rows[y] += "0"
			}
			rows[y] += g[y]
		}
	}
	return rows, nil
}

// MustText is Text for fixed strings known to have glyphs.
func MustText(s string) Bitmap {
	b, err := Text(s)
	if err != nil {
		panic(err)
	}
	return b
}

// Join places bitmaps side by side with gap empty columns between them.
// All parts must have the same height.
func Join(gap int, parts ...Bitmap) Bitmap {
	if len(parts) == 0 {
		return nil
	}
	rows := make(Bitmap, len(parts[0]))
	for i, p := range parts {
		for y := range rows {
			if i > 0 {
				rows[y] += strings.Repeat("0", gap)
			}
			rows[y] += p[y]
		}
	}
	return rows
}

// Stack places bitmaps top to bottom with gap empty rows between them,
// centering narrower parts (extra column goes on the right).
func Stack(gap int, parts ...Bitmap) Bitmap {
	w := 0
	for _, p := range parts {
		w = max(w, p.Width())
	}
	var rows Bitmap
	for i, p := range parts {
		if i > 0 {
			for range gap {
				rows = append(rows, strings.Repeat("0", w))
			}
		}
		left := (w - p.Width()) / 2
		right := w - p.Width() - left
		for _, r := range p {
			rows = append(rows, strings.Repeat("0", left)+r+strings.Repeat("0", right))
		}
	}
	return rows
}

// Scale enlarges every pixel to k×k.
func Scale(b Bitmap, k int) Bitmap {
	var rows Bitmap
	for _, r := range b {
		var sb strings.Builder
		for _, c := range r {
			sb.WriteString(strings.Repeat(string(c), k))
		}
		for range k {
			rows = append(rows, sb.String())
		}
	}
	return rows
}

// HalfBlocks renders the bitmap two pixel rows per text row using ▀ ▄ █,
// which keeps pixels square in a terminal. An odd last row is padded.
func HalfBlocks(b Bitmap) []string {
	rows := b
	if len(rows)%2 == 1 {
		rows = append(slicesClone(rows), strings.Repeat("0", b.Width()))
	}
	out := make([]string, 0, len(rows)/2)
	for y := 0; y < len(rows); y += 2 {
		var sb strings.Builder
		for x := range len(rows[y]) {
			top, bottom := rows[y][x] == '1', rows[y+1][x] == '1'
			switch {
			case top && bottom:
				sb.WriteRune('█')
			case top:
				sb.WriteRune('▀')
			case bottom:
				sb.WriteRune('▄')
			default:
				sb.WriteRune(' ')
			}
		}
		out = append(out, sb.String())
	}
	return out
}

func slicesClone(b Bitmap) Bitmap { return append(Bitmap(nil), b...) }
