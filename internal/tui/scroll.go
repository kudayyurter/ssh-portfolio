package tui

import "fmt"

// move applies a movement key to a cursor over n rows, page rows at a time,
// and keeps it in [0, n-1] (0 when n is 0).
func move(cursor, n int, key string, page int) int {
	switch key {
	case "up", "k":
		cursor--
	case "down", "j":
		cursor++
	case "pgup":
		cursor -= page
	case "pgdown", "space":
		cursor += page
	case "g", "home":
		cursor = 0
	case "G", "end":
		cursor = n - 1
	}
	return max(0, min(cursor, n-1))
}

// follow scrolls just enough to keep row cursor inside a rows-high window
// that starts at offset.
func follow(cursor, offset, rows int) int {
	switch {
	case cursor < offset:
		return cursor
	case cursor >= offset+rows:
		return cursor - rows + 1
	}
	return offset
}

// maxOffset is the last scroll offset for n lines shown rows at a time.
func maxOffset(n, rows int) int { return max(0, n-rows) }

// clampOffset keeps a scroll offset inside [0, maxOffset].
func clampOffset(offset, n, rows int) int { return max(0, min(offset, maxOffset(n, rows))) }

// marker is the footer's scroll position: empty when everything fits,
// "↓ more · 42%" while there is more below, "100%" at the end.
func marker(offset, n, rows int) string {
	last := maxOffset(n, rows)
	switch {
	case last == 0:
		return ""
	case offset >= last:
		return "100%"
	}
	return fmt.Sprintf("↓ more · %d%%", offset*100/last)
}
