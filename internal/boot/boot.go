// Package boot draws the startup animation: faint dots fly in from the
// screen edges and settle into "KUDAY YURTER" in the site's pixel font, while
// a tagline, progress bar and status line fill in underneath.
//
// Frame is a pure function of time and window size, so the animation can be
// tested frame by frame and survives resizes without restarting.
package boot

import (
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/kudayyurter/ssh-portfolio/internal/pixelfont"
)

// Duration is how long the animation runs before the shell takes over.
const Duration = 2100 * time.Millisecond

const (
	ms     = time.Millisecond
	flight = 400 * ms // how long each dot takes to reach its pixel
	below  = 5        // rows under the name: blank, tagline, blank, bar, status
)

// Options are the words drawn under the name.
type Options struct {
	Tagline  string
	Projects int // shown as "indexing N projects"
}

var (
	fullName = pixelfont.Join(4, pixelfont.MustText("KUDAY"), pixelfont.MustText("YURTER"))
	stacked  = pixelfont.Stack(1, pixelfont.MustText("KUDAY"), pixelfont.MustText("YURTER"))

	// layouts are tried in order; the first that fits the window wins.
	layouts = []struct {
		minW int
		art  []string
	}{
		{140, pixelfont.HalfBlocks(pixelfont.Scale(fullName, 2))}, // 136×7
		{72, pixelfont.HalfBlocks(fullName)},                      // 68×4
		{40, pixelfont.HalfBlocks(stacked)},                       // 35×8
	}
)

func pickLayout(w, h int) ([]string, bool) {
	if w < 40 || h < 12 {
		return nil, false
	}
	for _, l := range layouts {
		if w >= l.minW && h >= len(l.art)+below+2 {
			return l.art, true
		}
	}
	return nil, false
}

// Fits reports whether a w×h window is big enough for the animation.
func Fits(w, h int) bool {
	_, ok := pickLayout(w, h)
	return ok
}

type target struct {
	x, y int
	r    rune
}

// Frame draws the animation at time t for a w×h window: exactly h lines of
// w columns. Windows too small for any layout get a blank frame.
func Frame(t time.Duration, w, h int, o Options) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	c := newCanvas(w, h)
	art, ok := pickLayout(w, h)
	if !ok {
		return c.String()
	}
	artW, artH := utf8.RuneCountInString(art[0]), len(art)
	top, left := (h-(artH+below))/2, (w-artW)/2

	var targets []target
	for y, row := range art {
		for x, r := range []rune(row) {
			if r != ' ' {
				targets = append(targets, target{left + x, top + y, r})
			}
		}
	}

	n := len(targets)
	for i, tg := range targets {
		// Same scatter as the site's pixel reveal: index × 37 mod total.
		k := (i * 37) % n
		land := 400*ms + time.Duration(float64(600*ms)*float64(k)/float64(n))
		switch {
		case t >= land+120*ms:
			c.set(tg.x, tg.y, tg.r, plain)
		case t >= land+60*ms:
			c.set(tg.x, tg.y, '▪', faint)
		case t >= land:
			c.set(tg.x, tg.y, '·', faint)
		case t >= land-flight:
			u := float64(t-(land-flight)) / float64(flight)
			x, y := dot(i, tg, w, h, u)
			if c.empty(x, y) {
				c.set(x, y, '·', faint)
			}
		}
	}

	if t >= 200*ms {
		y := top + artH + 1
		c.text((w-utf8.RuneCountInString(o.Tagline))/2, y, o.Tagline, muted)

		barW := min(40, w-4)
		p := math.Min(1, float64(t-200*ms)/float64(1600*ms))
		filled := int(p*float64(barW) + 0.5)
		bx := (w - barW) / 2
		for i := range barW {
			if i < filled {
				c.set(bx+i, y+2, '━', plain)
			} else {
				c.set(bx+i, y+2, '─', faint)
			}
		}

		st := status(t, o.Projects)
		c.text((w-utf8.RuneCountInString(st))/2, y+3, st, faint)
	}
	return c.String()
}

func status(t time.Duration, projects int) string {
	switch {
	case t < 600*ms:
		return "mounting /home/guest"
	case t < 1100*ms:
		return fmt.Sprintf("indexing %d projects", projects)
	case t < 1600*ms:
		return "starting shell"
	}
	return "[ ok ] ready"
}

// dot is where dot i is at progress u∈[0,1): it starts on a screen edge
// chosen by a hash of i and follows a curved path (quadratic Bézier) to its
// pixel, slowing as it arrives (ease-out cubic).
func dot(i int, tg target, w, h int, u float64) (int, int) {
	hash := uint32(i)*2654435761 + 12345
	pos := float64((hash>>8)%1000) / 1000
	var sx, sy float64
	switch hash % 4 {
	case 0:
		sx, sy = pos*float64(w-1), 0
	case 1:
		sx, sy = float64(w-1), pos*float64(h-1)
	case 2:
		sx, sy = pos*float64(w-1), float64(h-1)
	default:
		sx, sy = 0, pos*float64(h-1)
	}
	tx, ty := float64(tg.x), float64(tg.y)
	bend := 0.35
	if hash&(1<<20) != 0 {
		bend = -bend
	}
	cx, cy := (sx+tx)/2-(ty-sy)*bend, (sy+ty)/2+(tx-sx)*bend

	e := 1 - math.Pow(1-u, 3)
	x := (1-e)*(1-e)*sx + 2*(1-e)*e*cx + e*e*tx
	y := (1-e)*(1-e)*sy + 2*(1-e)*e*cy + e*e*ty
	return int(math.Round(x)), int(math.Round(y))
}
