package tui

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/charmbracelet/x/ansi"
)

func TestListRows(t *testing.T) {
	fsys := testFS(t, fixture)
	work := node(t, fsys, "~/work")
	got := strings.Split(plain(strings.Join(listLines("Work", work, 0, 60), "\n")), "\n")
	want := []string{"Work", "", "▸ Acme  Engineer · 2025 — Present", "  Beta  Intern · 2024"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	got = strings.Split(plain(strings.Join(listLines("Work", work, 1, 60), "\n")), "\n")
	if got[2] != "  Acme  Engineer · 2025 — Present" || got[3] != "▸ Beta  Intern · 2024" {
		t.Fatalf("cursor 1: %q", got)
	}
	if listLen(work) != len(got) {
		t.Fatalf("listLen = %d, want %d", listLen(work), len(got))
	}
}

func TestListTruncatesToWidth(t *testing.T) {
	fsys := testFS(t, fixture)
	for _, l := range listLines("Work", node(t, fsys, "~/work"), 0, 20) {
		if w := ansi.StringWidth(l); w > 20 {
			t.Fatalf("row wider than 20 (%d): %q", w, ansi.Strip(l))
		}
	}
}

func TestEmptyList(t *testing.T) {
	fsys := testFS(t, fstest.MapFS{
		"about.md": {Data: []byte("---\ntitle: About\nsummary: x\n---\nHi.\n")},
		"work":     {Mode: fs.ModeDir},
	})
	work := node(t, fsys, "~/work")
	got := plain(strings.Join(listLines("Work", work, 0, 60), "\n"))
	if got != "Work\n\nnothing here yet" || listLen(work) != 3 {
		t.Fatalf("got %q (len %d)", got, listLen(work))
	}
}
