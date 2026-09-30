package shell

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/kudayyurter/ssh-portfolio/content"
	"github.com/kudayyurter/ssh-portfolio/internal/vfs"
)

// newTestSession uses a small fixed tree so tests don't break when the real
// portfolio content changes.
func newTestSession(t *testing.T) *Session {
	t.Helper()
	fsys, err := vfs.New(fstest.MapFS{
		"about.md":            {Data: []byte("---\ntitle: About\nsummary: Who I am\norder: 1\n---\nHello **there**.\n")},
		"contact.md":          {Data: []byte("---\ntitle: Contact\nsummary: How to reach me\nlink: mailto:me@example.com\norder: 2\n---\nEmail me.\n")},
		"work/acme.md":        {Data: []byte("---\ntitle: Acme\nsummary: Engineer\ndate: 2025 — Present\norder: 1\n---\nBuilt things.\n")},
		"work/beta.md":        {Data: []byte("---\ntitle: Beta\nsummary: Intern\ndate: 2024\norder: 2\n---\nLearned things.\n")},
		"projects/kessler.md": {Data: []byte("---\ntitle: Kessler\nsummary: Orbit globe\nstack: Go · Three.js\nlink: https://kessler.example.com\n---\nA globe.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := content.Profile{
		Name:    "Test Person",
		Tagline: "tester of things",
		Role:    "Tester at Acme",
		School:  "Test University",
		Degree:  "B.S. Testing",
		Stack:   []string{"Go", "SQL"},
		Links:   []content.Link{{Label: "Web", URL: "https://example.com"}},
	}
	return New(fsys, profile, Options{
		Width:       80,
		Interactive: true,
		Now:         func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) },
	})
}

// plain strips styling and trailing spaces so tests compare visible text.
func plain(r Result) string {
	lines := strings.Split(ansi.Strip(r.Output), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func run(t *testing.T, s *Session, line string) (string, Result) {
	t.Helper()
	r := s.Run(line)
	return plain(r), r
}
