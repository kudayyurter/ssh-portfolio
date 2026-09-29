package tui

import "testing"

func TestMove(t *testing.T) {
	cases := []struct {
		cursor, n int
		key       string
		want      int
	}{
		{0, 5, "down", 1}, {0, 5, "j", 1}, {3, 5, "up", 2}, {3, 5, "k", 2},
		{0, 5, "up", 0}, {4, 5, "down", 4}, // stays in bounds
		{0, 50, "pgdown", 10}, {0, 50, "space", 10}, {25, 50, "pgup", 15},
		{3, 5, "g", 0}, {3, 5, "home", 0}, {1, 5, "G", 4}, {1, 5, "end", 4},
		{0, 0, "down", 0}, {0, 0, "G", 0}, // empty
		{2, 5, "x", 2}, // unrelated key
	}
	for _, c := range cases {
		if got := move(c.cursor, c.n, c.key, 10); got != c.want {
			t.Errorf("move(%d, %d, %q) = %d, want %d", c.cursor, c.n, c.key, got, c.want)
		}
	}
}

func TestFollowAndClamp(t *testing.T) {
	if got := follow(12, 0, 10); got != 3 {
		t.Errorf("follow below = %d, want 3", got)
	}
	if got := follow(2, 5, 10); got != 2 {
		t.Errorf("follow above = %d, want 2", got)
	}
	if got := follow(7, 5, 10); got != 5 {
		t.Errorf("follow inside = %d, want 5", got)
	}
	if got := clampOffset(40, 30, 10); got != 20 {
		t.Errorf("clamp high = %d, want 20", got)
	}
	if got := clampOffset(5, 3, 10); got != 0 {
		t.Errorf("clamp short = %d, want 0", got)
	}
	if got := maxOffset(3, 10); got != 0 {
		t.Errorf("maxOffset short = %d, want 0", got)
	}
}

func TestMarker(t *testing.T) {
	if got := marker(0, 5, 10); got != "" {
		t.Errorf("fits: %q", got)
	}
	if got := marker(0, 30, 10); got != "↓ more · 0%" {
		t.Errorf("top: %q", got)
	}
	if got := marker(10, 30, 10); got != "↓ more · 50%" {
		t.Errorf("middle: %q", got)
	}
	if got := marker(20, 30, 10); got != "100%" {
		t.Errorf("end: %q", got)
	}
}
