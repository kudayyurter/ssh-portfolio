package shell

import "github.com/kudayyurter/termfolio/internal/style"

func notFound(name string) Result {
	out := style.Err.Render(name + ": command not found")
	if m := closest(name); m != "" {
		out += "\n" + style.Faint.Render("did you mean "+m+"?")
	}
	return Result{Output: out, Code: 127}
}

// closest is the visible command nearest to name, if it is close enough to
// be a typo: at most 2 edits, and at most half of a short name.
func closest(name string) string {
	best, bestD := "", 3
	for _, c := range visibleCommands() {
		d := distance(name, c)
		if d < bestD {
			best, bestD = c, d
		}
	}
	if best == "" || bestD > len([]rune(name))/2+1 {
		return ""
	}
	return best
}

// distance is the Damerau–Levenshtein distance (swapped neighbors count as
// one edit, so "sl" is one step from "ls").
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range rb {
		d[0][j+1] = j + 1
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}
