package tui

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/charmbracelet/x/ansi"
	"github.com/kudayyurter/termfolio/content"
	"github.com/kudayyurter/termfolio/internal/vfs"
)

// fixture is a small fixed tree so tests don't break when the real content
// changes. It has no stack.md, so the Stack menu entry is left out.
var fixture = fstest.MapFS{
	"about.md":            {Data: []byte("---\ntitle: About\nsummary: Who I am\norder: 1\n---\nHello **there**.\n")},
	"contact.md":          {Data: []byte("---\ntitle: Contact\nsummary: How to reach me\norder: 2\n---\n- Email: me@example.com\n- GitHub: https://github.com/tester\n")},
	"work/acme.md":        {Data: []byte("---\ntitle: Acme\nsummary: Engineer\ndate: 2025 — Present\norder: 1\n---\nBuilt things.\n")},
	"work/beta.md":        {Data: []byte("---\ntitle: Beta\nsummary: Intern\ndate: 2024\norder: 2\n---\nLearned things.\n")},
	"projects/kessler.md": {Data: []byte("---\ntitle: Kessler\nsummary: Orbit globe\nstack: Go · Three.js\nlink: https://kessler.example.com\n---\nA globe.\n")},
}

var testProfile = content.Profile{
	Name:    "Test Person",
	Tagline: "tester of things",
	Links: []content.Link{
		{Label: "Web", URL: "https://example.com"},
		{Label: "GitHub", URL: "https://github.com/tester"},
		{Label: "LinkedIn", URL: "https://www.linkedin.com/in/tester/"},
		{Label: "Email", URL: "mailto:me@example.com"},
	},
}

func testFS(t *testing.T, files fstest.MapFS) *vfs.FS {
	t.Helper()
	fsys, err := vfs.New(files)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func node(t *testing.T, fsys *vfs.FS, p string) *vfs.Node {
	t.Helper()
	n, err := fsys.Resolve(vfs.Home, p)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// plain strips styling and trailing spaces so tests compare visible text.
func plain(s string) string {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}
