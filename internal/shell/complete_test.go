package shell

import (
	"slices"
	"testing"
)

func TestComplete(t *testing.T) {
	s := newTestSession(t)
	cases := []struct {
		line, want string
		cands      []string
	}{
		{"ca", "cat ", nil},
		{"c", "c", []string{"cat", "cd", "clear"}},
		{"ne", "neofetch ", nil},
		{"zz", "zz", nil},
		{"cat ab", "cat about.md ", nil},
		{"cd w", "cd work/", nil},
		{"cat work/a", "cat work/acme.md ", nil},
		{"cat work/", "cat work/", []string{"acme.md", "beta.md"}},
		{"ls ~/pro", "ls ~/projects/", nil},
		{"ls /h", "ls /home/", nil},
		{"cat ", "cat ", []string{"projects/", "work/", "about.md", "contact.md"}},
		{"cat nope/x", "cat nope/x", nil},
		{"cat about.md/x", "cat about.md/x", nil},
		{"cat about.md c", "cat about.md contact.md ", nil},
		{"sud", "sud", nil}, // hidden commands are not offered
	}
	for _, c := range cases {
		got, cands := s.Complete(c.line)
		if got != c.want || !slices.Equal(cands, c.cands) {
			t.Errorf("Complete(%q) = %q %q, want %q %q", c.line, got, cands, c.want, c.cands)
		}
	}
}

func TestCompleteUsesCwd(t *testing.T) {
	s := newTestSession(t)
	s.Run("cd work")
	if got, _ := s.Complete("cat b"); got != "cat beta.md " {
		t.Fatalf("Complete in work = %q", got)
	}
	if got, _ := s.Complete("cat ../ab"); got != "cat ../about.md " {
		t.Fatalf("Complete ../ = %q", got)
	}
}
