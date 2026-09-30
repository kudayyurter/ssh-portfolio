package shell

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kudayyurter/termfolio/content"
	"github.com/kudayyurter/termfolio/internal/vfs"
)

func TestCat(t *testing.T) {
	s := newTestSession(t)
	got, r := run(t, s, "cat about.md")
	if got != "About\n\nHello there." || r.Code != 0 {
		t.Errorf("cat about.md = %q (%d)", got, r.Code)
	}
	got, _ = run(t, s, "cat projects/kessler.md")
	want := "Kessler\nGo · Three.js\nkessler.example.com\n\nA globe."
	if got != want {
		t.Errorf("cat kessler =\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(s.Run("cat projects/kessler.md").Output, "\x1b]8;;https://kessler.example.com") {
		t.Error("link is not an OSC 8 hyperlink")
	}
	got, _ = run(t, s, "cat work/acme.md")
	if !strings.HasPrefix(got, "Acme\n2025 — Present\n\n") {
		t.Errorf("cat acme = %q", got)
	}
}

func TestCatErrorsAndAliases(t *testing.T) {
	s := newTestSession(t)
	cases := []struct {
		line, want string
		code       int
	}{
		{"cat", "cat: missing file operand", 1},
		{"cat work", "cat: work: Is a directory", 1},
		{"cat nope", "cat: nope: No such file or directory", 1},
		{"cat about.md/", "cat: about.md/: Not a directory", 1},
		{"cat about.md nope", "About\n\nHello there.\n\ncat: nope: No such file or directory", 1},
		{"less about.md", "About\n\nHello there.", 0},
		{"more work", "more: work: Is a directory", 1},
		{"cat -n about.md", "cat: invalid option -- 'n'", 1},
	}
	for _, c := range cases {
		if got, r := run(t, s, c.line); got != c.want || r.Code != c.code {
			t.Errorf("%q = %q (%d), want %q (%d)", c.line, got, r.Code, c.want, c.code)
		}
	}
}

func TestOpen(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "open projects/kessler.md"); got != "https://kessler.example.com" || r.Code != 0 {
		t.Errorf("open = %q (%d)", got, r.Code)
	}
	cases := map[string]string{
		"open about.md": "open: about.md: no link",
		"open work":     "open: work: no link",
		"open nope":     "open: nope: No such file or directory",
		"open":          "usage: open <file>",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestWhoami(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, "whoami"); got != "guest — visiting Test Person · tester of things" {
		t.Errorf("whoami = %q", got)
	}
}

func TestNeofetch(t *testing.T) {
	s := newTestSession(t)
	got, _ := run(t, s, "neofetch")
	for _, want := range []string{"guest@kuday", "Tester at Acme", "Go · SQL", "example.com", "██      ██  ██      ██"} {
		if !strings.Contains(got, want) {
			t.Errorf("neofetch missing %q:\n%s", want, got)
		}
	}
	for _, l := range strings.Split(s.Run("neofetch").Output, "\n") {
		if strings.HasSuffix(l, " ") {
			t.Errorf("neofetch line has trailing padding: %q", l)
		}
	}
	s.SetWidth(50)
	got, _ = run(t, s, "neofetch")
	if strings.Contains(got, "██") || !strings.Contains(got, "Test University") {
		t.Errorf("narrow neofetch should drop the logo:\n%s", got)
	}
}

func TestHelpListsVisibleCommandsOnly(t *testing.T) {
	s := newTestSession(t)
	got, _ := run(t, s, "help")
	for _, want := range []string{"Moving around", "Reading", "About", "Utilities", "ls [-la] [path]", "cat <file>", "neofetch", "boot", "PgUp/PgDn scroll"} {
		if !strings.Contains(got, want) {
			t.Errorf("help missing %q", want)
		}
	}
	for _, hidden := range []string{"sudo", "vim", "rm", "less", "logout"} {
		if strings.Contains(got, "  "+hidden+" ") || strings.Contains(got, "\n  "+hidden) {
			t.Errorf("help shows hidden command %q", hidden)
		}
	}
}

func TestUtilities(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, `echo hello   "big world"`); got != "hello big world" {
		t.Errorf("echo = %q", got)
	}
	if got, _ := run(t, s, "date"); got != "Mon Sep 28 12:00:00 UTC 2026" {
		t.Errorf("date = %q", got)
	}
	if got, _ := run(t, s, "history"); got != "    1  echo hello   \"big world\"\n    2  date\n    3  history" {
		t.Errorf("history = %q", got)
	}
	if r := s.Run("clear"); r.Action != ActionClear || r.Output != "" {
		t.Errorf("clear = %+v", r)
	}
	for _, line := range []string{"exit", "logout"} {
		if r := s.Run(line); r.Action != ActionExit || r.Output != "logout" {
			t.Errorf("%s = %+v", line, r)
		}
	}
	if r := s.Run("boot"); r.Action != ActionBoot {
		t.Errorf("boot = %+v", r)
	}
}

func TestBootNeedsATerminal(t *testing.T) {
	fsys, err := vfs.New(fstest.MapFS{"a.md": {Data: []byte("---\ntitle: A\nsummary: B\n---\n")}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(fsys, content.Profile{}, Options{})
	if r := s.Run("boot"); r.Code != 1 || r.Action != ActionNone {
		t.Errorf("non-interactive boot = %+v", r)
	}
}

func TestEasterEggs(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"sudo rm -rf /": "guest is not in the sudoers file. This incident will be reported.",
		"rm about.md":   "rm: read-only file system (nice try)",
		"mkdir x":       "mkdir: read-only file system (nice try)",
		"vim about.md":  "vim: read-only file system — try cat about.md",
		"nano":          "nano: read-only file system — try cat <file>",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestSuggestionsIncludeLaterCommands(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, "cta"); got != "cta: command not found\ndid you mean cat?" {
		t.Errorf("cta = %q", got)
	}
	if got, _ := run(t, s, "hlep"); got != "hlep: command not found\ndid you mean help?" {
		t.Errorf("hlep = %q", got)
	}
}

func TestCatLimitsOperands(t *testing.T) {
	s := newTestSession(t)
	line := "cat" + strings.Repeat(" about.md", 11)
	if got, r := run(t, s, line); got != "cat: too many files (at most 10)" || r.Code != 1 {
		t.Fatalf("11 files = %q (%d)", got, r.Code)
	}
	if _, r := run(t, s, "cat"+strings.Repeat(" about.md", 10)); r.Code != 0 {
		t.Fatalf("10 files failed: %d", r.Code)
	}
}

func TestHistoryIsCapped(t *testing.T) {
	s := newTestSession(t)
	for i := range 600 {
		s.Run(fmt.Sprintf("echo %d", i))
	}
	h := s.History()
	if len(h) != 500 || h[0] != "echo 100" || h[499] != "echo 599" {
		t.Fatalf("history len %d, first %q, last %q", len(h), h[0], h[len(h)-1])
	}
}
