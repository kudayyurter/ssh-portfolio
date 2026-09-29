package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
)

func testMenu(t *testing.T) []menuItem {
	fsys := testFS(t, fixture)
	return []menuItem{
		{"About", node(t, fsys, "~/about.md")},
		{"Work", node(t, fsys, "~/work")},
		{"Projects", node(t, fsys, "~/projects")},
		{"Contact", node(t, fsys, "~/contact.md")},
	}
}

// menuOrder is the menu labels in the order they appear on screen.
func menuOrder(screen string) []string {
	var got []string
	for _, l := range strings.Split(screen, "\n") {
		s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "▸"))
		switch s {
		case "About", "Work", "Projects", "Stack", "Contact":
			got = append(got, s)
		}
	}
	return got
}

func TestHomeWide(t *testing.T) {
	raw := strings.Join(homeLines(testProfile, testMenu(t), 0, 100, 30), "\n")
	screen := plain(raw)
	if !strings.Contains(screen, "█") {
		t.Fatal("wide home should show the monogram")
	}
	for _, s := range []string{"Test Person", "tester of things", " web", " github", " linkedin", " email", "▸ About"} {
		if !strings.Contains(screen, s) {
			t.Errorf("missing %q in\n%s", s, screen)
		}
	}
	for _, l := range testProfile.Links {
		if !strings.Contains(raw, ansi.SetHyperlink(l.URL)) {
			t.Errorf("%s is not a hyperlink", l.URL)
		}
	}
	if got := strings.Join(menuOrder(screen), " "); got != "About Work Projects Contact" {
		t.Fatalf("menu order = %q", got)
	}
}

func TestHomeNarrowStacksLinks(t *testing.T) {
	screen := plain(strings.Join(homeLines(testProfile, testMenu(t), 2, 60, 24), "\n"))
	if strings.Contains(screen, "█") {
		t.Fatal("narrow home should hide the monogram")
	}
	for _, l := range strings.Split(screen, "\n") {
		if strings.Contains(l, "web") && strings.Contains(l, "github") {
			t.Fatalf("links should be one per row when narrow: %q", l)
		}
	}
	if !strings.Contains(screen, "▸ Projects") || !strings.Contains(screen, " email") {
		t.Fatalf("screen:\n%s", screen)
	}
}

func TestHomeDropsLinksThenNameToFit(t *testing.T) {
	screen := plain(strings.Join(homeLines(testProfile, testMenu(t), 0, 30, 10), "\n"))
	if !strings.Contains(screen, "Test Person") || strings.Contains(screen, "github") {
		t.Fatalf("30×10 keeps the name, drops the links:\n%s", screen)
	}
	screen = plain(strings.Join(homeLines(testProfile, testMenu(t), 0, 20, 8), "\n"))
	if strings.Contains(screen, "Test Person") || len(menuOrder(screen)) != 4 {
		t.Fatalf("20×8 shows only the menu:\n%s", screen)
	}
}

func TestHomeFitsItsRoom(t *testing.T) {
	for _, s := range [][2]int{{20, 8}, {30, 10}, {60, 24}, {72, 16}, {100, 30}, {512, 256}} {
		lines := homeLines(testProfile, testMenu(t), 0, s[0], s[1])
		if len(lines) > s[1]-2 {
			t.Errorf("%dx%d: %d rows, room for %d", s[0], s[1], len(lines), s[1]-2)
		}
		for _, l := range lines {
			if ansi.StringWidth(l) > s[0] {
				t.Errorf("%dx%d: row too wide: %q", s[0], s[1], ansi.Strip(l))
			}
		}
	}
}

// Review focus 3: odd profiles still render.
func TestLinkGridUnknownLabelAndNoLinks(t *testing.T) {
	rows := linkGrid([]content.Link{{Label: "Blog", URL: "https://blog.example.com"}}, 2)
	if len(rows) != 1 || plain(rows[0]) != "• blog" {
		t.Fatalf("unknown label: %q", rows)
	}
	if rows := linkGrid(nil, 2); len(rows) != 0 {
		t.Fatalf("no links: %q", rows)
	}
	p := testProfile
	p.Links = nil
	screen := plain(strings.Join(homeLines(p, testMenu(t), 0, 100, 30), "\n"))
	if !strings.Contains(screen, "Test Person") || len(menuOrder(screen)) != 4 {
		t.Fatalf("no links:\n%s", screen)
	}
}
