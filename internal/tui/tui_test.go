package tui

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func key(s string) tea.KeyPressMsg {
	codes := map[string]rune{
		"enter": tea.KeyEnter, "esc": tea.KeyEscape, "backspace": tea.KeyBackspace,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd,
	}
	switch {
	case s == "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case s == "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case codes[s] != 0:
		return tea.KeyPressMsg{Code: codes[s]}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

func press(m Model, keys ...string) (Model, Result) {
	var res Result
	for _, k := range keys {
		m, res = m.Update(key(k))
	}
	return m, res
}

func TestKeyHelperMatchesUpdateNames(t *testing.T) {
	for _, s := range []string{"enter", "esc", "backspace", "pgdown", "space", "ctrl+c", "j", "G"} {
		if got := key(s).String(); got != s {
			t.Errorf("key(%q).String() = %q", s, got)
		}
	}
}

func TestMenuSkipsMissingPaths(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 80, 24)
	var labels []string
	for _, it := range m.menu {
		labels = append(labels, it.label)
	}
	if got := strings.Join(labels, " "); got != "About Work Projects Contact" {
		t.Fatalf("menu = %q (fixture has no stack.md)", got)
	}
}

func TestNavigation(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 80, 24)
	m, res := press(m, "j", "enter")
	if res.Opened != vfs.Home+"/work" || len(m.stack) != 2 {
		t.Fatalf("open Work: %+v depth %d", res, len(m.stack))
	}
	m, res = press(m, "enter")
	if res.Opened != vfs.Home+"/work/acme.md" || len(m.stack) != 3 || m.stack[2].kind != kindDetail {
		t.Fatalf("open Acme: %+v depth %d", res, len(m.stack))
	}
	m, _ = press(m, "esc")
	m, _ = press(m, "backspace")
	if len(m.stack) != 1 || m.stack[0].cursor != 1 {
		t.Fatalf("back twice: depth %d cursor %d", len(m.stack), m.stack[0].cursor)
	}
	m, res = press(m, "esc")
	if len(m.stack) != 1 || res.Quit {
		t.Fatalf("back on home: depth %d %+v", len(m.stack), res)
	}
	for _, k := range []string{"left", "h"} {
		m, _ = press(m, "right") // opens Work
		if len(m.stack) != 2 {
			t.Fatalf("right should open")
		}
		m, _ = press(m, k)
		if len(m.stack) != 1 {
			t.Fatalf("%s should go back", k)
		}
	}
	m, res = press(m, "l")
	if res.Opened != vfs.Home+"/work" {
		t.Fatalf("l should open: %+v", res)
	}
}

func TestQuitFromEveryDepth(t *testing.T) {
	for _, path := range [][]string{{}, {"j", "enter"}, {"j", "enter", "enter"}} {
		for _, q := range []string{"q", "ctrl+c"} {
			m := New(testFS(t, fixture), testProfile, 80, 24)
			m, _ = press(m, path...)
			if _, res := press(m, q); !res.Quit {
				t.Errorf("%s after %q did not quit", q, path)
			}
		}
	}
}

func TestHomeCursorStaysInBounds(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 80, 24)
	steps := []struct {
		key  string
		want int
	}{{"k", 0}, {"G", 3}, {"j", 3}, {"g", 0}, {"pgdown", 3}, {"up", 2}, {"home", 0}, {"end", 3}}
	for _, s := range steps {
		m, _ = press(m, s.key)
		if m.stack[0].cursor != s.want {
			t.Fatalf("after %s cursor = %d, want %d", s.key, m.stack[0].cursor, s.want)
		}
	}
}

func TestListKeys(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 80, 24)
	m, _ = press(m, "j", "enter", "G")
	if m.stack[1].cursor != 1 {
		t.Fatalf("G: cursor %d", m.stack[1].cursor)
	}
	m, _ = press(m, "j")
	if m.stack[1].cursor != 1 {
		t.Fatalf("j at end: cursor %d", m.stack[1].cursor)
	}
	m, res := press(m, "pgup", "enter")
	if res.Opened != vfs.Home+"/work/acme.md" {
		t.Fatalf("pgup then enter: %+v", res)
	}
}

// longFS has an about page far longer than any window.
func longFS(t *testing.T) *vfs.FS {
	var b strings.Builder
	for i := range 60 {
		fmt.Fprintf(&b, "line %02d\n\n", i)
	}
	return testFS(t, fstest.MapFS{"about.md": {Data: []byte("---\ntitle: About\nsummary: x\n---\n" + b.String())}})
}

func TestDetailScroll(t *testing.T) {
	m := New(longFS(t), testProfile, 80, 24)
	m, _ = press(m, "enter")
	n, rows := len(m.stack[1].lines), m.rows()
	if rows != 21 || n <= rows {
		t.Fatalf("rows %d, lines %d", rows, n)
	}
	top := func() int { return m.stack[1].offset }
	m, _ = press(m, "G")
	if top() != n-rows || !strings.Contains(plain(m.View()), "100%") {
		t.Fatalf("G: offset %d want %d\n%s", top(), n-rows, plain(m.View()))
	}
	m, _ = press(m, "g")
	if top() != 0 || !strings.Contains(plain(m.View()), "↓ more · 0%") {
		t.Fatalf("g: offset %d", top())
	}
	m, _ = press(m, "pgdown")
	m, _ = press(m, "space")
	m, _ = press(m, "k")
	if top() != 2*rows-1 {
		t.Fatalf("pgdown space k: offset %d want %d", top(), 2*rows-1)
	}
	m, _ = press(m, "down", "j", "end", "pgdown")
	if top() != n-rows {
		t.Fatalf("past the end: offset %d", top())
	}
	if _, res := press(m, "enter"); res.Opened != "" {
		t.Fatalf("enter on a detail page opened %q", res.Opened)
	}
}

func TestResizeClampsScroll(t *testing.T) {
	m := New(longFS(t), testProfile, 80, 24)
	m, _ = press(m, "enter", "G")
	m.SetSize(80, 60)
	if want := maxOffset(len(m.stack[1].lines), 57); m.stack[1].offset != want {
		t.Fatalf("offset %d, want %d", m.stack[1].offset, want)
	}
	m.SetSize(40, 24)
	for _, l := range m.stack[1].lines {
		if ansi.StringWidth(l) > 38 {
			t.Fatalf("not re-rendered at 38 columns: %q", ansi.Strip(l))
		}
	}
}

// Review focus 5: the selected row stays visible when the window shrinks.
func TestResizeKeepsListCursorVisible(t *testing.T) {
	files := fstest.MapFS{}
	for i := range 30 {
		files[fmt.Sprintf("work/e%02d.md", i)] = &fstest.MapFile{Data: []byte(fmt.Sprintf("---\ntitle: Entry %02d\nsummary: s\norder: %d\n---\nx\n", i, i+1))}
	}
	m := New(testFS(t, files), testProfile, 80, 40)
	m, _ = press(m, "enter", "G")
	m.SetSize(80, 10)
	if !strings.Contains(plain(m.View()), "▸ Entry 29") {
		t.Fatalf("cursor row scrolled away:\n%s", plain(m.View()))
	}
	m, _ = press(m, "g")
	if !strings.Contains(plain(m.View()), "Work") {
		t.Fatalf("g should show the list title again:\n%s", plain(m.View()))
	}
}

func TestTinyWindow(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 30, 7)
	if m.View() != "make the window a bit bigger" {
		t.Fatalf("30×7: %q", m.View())
	}
	m.SetSize(19, 24)
	if m.View() != "make the window a b" {
		t.Fatalf("19×24: %q", m.View())
	}
	m.SetSize(80, 24)
	if !strings.Contains(plain(m.View()), "About") {
		t.Fatal("page should come back when the window grows")
	}
}

func TestEmptySection(t *testing.T) {
	fsys := testFS(t, fstest.MapFS{
		"about.md": {Data: []byte("---\ntitle: About\nsummary: x\n---\nHi.\n")},
		"work":     {Mode: fs.ModeDir},
	})
	m := New(fsys, testProfile, 80, 24)
	m, _ = press(m, "j", "enter")
	if !strings.Contains(plain(m.View()), "nothing here yet") {
		t.Fatalf("empty list:\n%s", plain(m.View()))
	}
	if _, res := press(m, "enter"); res.Opened != "" {
		t.Fatalf("enter on an empty list opened %q", res.Opened)
	}
}

func TestFrame(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 80, 24)
	m, _ = press(m, "j", "j", "enter")
	screen := strings.Split(plain(m.View()), "\n")
	if screen[0] != " ‹ Home  esc" || !strings.HasPrefix(screen[1], " ───") {
		t.Fatalf("top bar: %q", screen[:2])
	}
	if screen[len(screen)-1] != " ↑↓ move · ↵ open · esc back · q quit" {
		t.Fatalf("list footer: %q", screen[len(screen)-1])
	}
	m, _ = press(m, "enter")
	screen = strings.Split(plain(m.View()), "\n")
	if screen[0] != " ‹ Projects  esc" || screen[2] != " Kessler" {
		t.Fatalf("detail top: %q", screen[:3])
	}
	if screen[len(screen)-1] != " ↑↓ scroll · esc back · q quit" {
		t.Fatalf("detail footer: %q", screen[len(screen)-1])
	}
}

// Review focus 1: rows never wrap and the view is always the window height.
func TestViewAlwaysFitsTheWindow(t *testing.T) {
	p := testProfile
	p.Tagline = strings.Repeat("a very long tagline ", 10)
	sizes := [][2]int{{20, 8}, {30, 10}, {60, 24}, {80, 24}, {100, 30}, {512, 256}}
	for _, keys := range [][]string{{}, {"j", "enter"}, {"j", "j", "enter", "enter"}, {"G", "enter"}} {
		for _, s := range sizes {
			m := New(testFS(t, fixture), p, s[0], s[1])
			m, _ = press(m, keys...)
			rows := strings.Split(m.View(), "\n")
			if len(rows) != s[1] {
				t.Errorf("%v at %dx%d: %d rows", keys, s[0], s[1], len(rows))
			}
			for _, r := range rows {
				if ansi.StringWidth(r) > s[0] {
					t.Errorf("%v at %dx%d: row too wide: %q", keys, s[0], s[1], ansi.Strip(r))
				}
			}
		}
	}
}

// Review focus 2: before the first window size the TUI acts as 80×24.
func TestZeroSizeActsLike80x24(t *testing.T) {
	m := New(testFS(t, fixture), testProfile, 0, 0)
	m, _ = press(m, "j", "enter")
	if rows := strings.Split(m.View(), "\n"); len(rows) != 24 || !strings.Contains(plain(m.View()), "Acme") {
		t.Fatalf("%d rows:\n%s", len(rows), plain(m.View()))
	}
}
