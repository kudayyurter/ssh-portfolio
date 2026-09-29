package tui

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/charmbracelet/x/ansi"
)

func TestDetailHeader(t *testing.T) {
	fsys := testFS(t, fixture)
	got := detailLines(node(t, fsys, "~/projects/kessler.md"), 60)
	lines := strings.Split(plain(strings.Join(got, "\n")), "\n")
	want := []string{"Kessler", "Go · Three.js", "kessler.example.com", ""}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d = %q, want %q\nall: %q", i, lines[i], w, lines)
		}
	}
	if !strings.Contains(plain(strings.Join(got, "\n")), "A globe.") {
		t.Fatalf("body missing: %q", lines)
	}
	if !strings.Contains(strings.Join(got, "\n"), ansi.SetHyperlink("https://kessler.example.com")) {
		t.Fatal("project link is not a hyperlink")
	}
}

func TestDetailWithoutMetaStartsWithBody(t *testing.T) {
	fsys := testFS(t, fixture)
	lines := strings.Split(plain(strings.Join(detailLines(node(t, fsys, "~/about.md"), 60), "\n")), "\n")
	if lines[0] != "About" || lines[1] != "" || lines[2] != "Hello there." {
		t.Fatalf("lines = %q", lines)
	}
}

// Contact's addresses are clickable because glamour links them.
func TestDetailBodyLinksAreHyperlinks(t *testing.T) {
	fsys := testFS(t, fixture)
	raw := strings.Join(detailLines(node(t, fsys, "~/contact.md"), 60), "\n")
	for _, uri := range []string{"mailto:me@example.com", "https://github.com/tester"} {
		if !strings.Contains(raw, ";"+uri+"\x07") {
			t.Errorf("no hyperlink to %s in %q", uri, raw)
		}
	}
}

func TestDetailWrapsAtWidth(t *testing.T) {
	long := strings.Repeat("word ", 80)
	fsys := testFS(t, fstest.MapFS{"about.md": {Data: []byte("---\ntitle: About\nsummary: x\n---\n" + long + "\n")}})
	for _, l := range detailLines(node(t, fsys, "~/about.md"), 40) {
		if w := ansi.StringWidth(l); w > 40 {
			t.Fatalf("line wider than 40 (%d): %q", w, ansi.Strip(l))
		}
	}
}
