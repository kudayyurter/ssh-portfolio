package shell

import (
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	s := newTestSession(t)
	cases := []struct{ line, want string }{
		{"ls", "projects/  work/  about.md  contact.md"},
		{"ls work", "acme.md  beta.md"},
		{"ls -a work", "./  ../  acme.md  beta.md"},
		{"ls about.md", "about.md"},
		{"ls /", "home/"},
		{"ls -l", strings.Join([]string{
			"drwxr-xr-x  projects/   1 item",
			"drwxr-xr-x  work/       2 items",
			"-rw-r--r--  about.md    Who I am",
			"-rw-r--r--  contact.md  How to reach me",
		}, "\n")},
		{"ls -la work", strings.Join([]string{
			"drwxr-xr-x  ./",
			"drwxr-xr-x  ../",
			"-rw-r--r--  acme.md  Engineer · 2025 — Present",
			"-rw-r--r--  beta.md  Intern · 2024",
		}, "\n")},
		{"ls work projects", "work:\nacme.md  beta.md\n\nprojects:\nkessler.md"},
	}
	for _, c := range cases {
		got, r := run(t, s, c.line)
		if got != c.want || r.Code != 0 {
			t.Errorf("%q (code %d) =\n%s\nwant\n%s", c.line, r.Code, got, c.want)
		}
	}
}

func TestLsWrapsToWidth(t *testing.T) {
	s := newTestSession(t)
	s.SetWidth(20)
	got, _ := run(t, s, "ls")
	for _, l := range strings.Split(got, "\n") {
		if len(l) > 20 {
			t.Fatalf("line wider than 20: %q", l)
		}
	}
}

func TestLsErrors(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "ls nope"); got != "ls: nope: No such file or directory" || r.Code != 1 {
		t.Errorf("ls nope = %q (%d)", got, r.Code)
	}
	if got, r := run(t, s, "ls -z"); got != "ls: invalid option -- 'z'" || r.Code != 1 {
		t.Errorf("ls -z = %q (%d)", got, r.Code)
	}
}

func TestCdAndPwd(t *testing.T) {
	s := newTestSession(t)
	steps := []struct{ line, out, cwd string }{
		{"cd work", "", "/home/guest/work"},
		{"pwd", "/home/guest/work", "/home/guest/work"},
		{"cd ..", "", "/home/guest"},
		{"cd -", "~/work", "/home/guest/work"},
		{"cd", "", "/home/guest"},
		{"cd /", "", "/"},
		{"cd ~/projects", "", "/home/guest/projects"},
		{"cd ../../../../..", "", "/"},
		{"cd /home", "", "/home"},
	}
	for _, st := range steps {
		got, r := run(t, s, st.line)
		if got != st.out || s.Cwd() != st.cwd || r.Code != 0 {
			t.Errorf("%q: out %q cwd %q code %d; want out %q cwd %q", st.line, got, s.Cwd(), r.Code, st.out, st.cwd)
		}
	}
}

func TestCdErrors(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"cd nope":     "cd: nope: No such file or directory",
		"cd about.md": "cd: about.md: Not a directory",
		"cd a b":      "cd: too many arguments",
		"cd -":        "cd: OLDPWD not set",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
	if s.Cwd() != "/home/guest" {
		t.Fatalf("failed cd changed cwd to %q", s.Cwd())
	}
}

func TestPromptFollowsCwd(t *testing.T) {
	s := newTestSession(t)
	if got := plain(Result{Output: s.Prompt()}); got != "guest@kuday:~$" {
		t.Fatalf("prompt = %q", got)
	}
	s.Run("cd work")
	if got := plain(Result{Output: s.Prompt()}); got != "guest@kuday:~/work$" {
		t.Fatalf("prompt = %q", got)
	}
}

func TestTree(t *testing.T) {
	s := newTestSession(t)
	want := strings.Join([]string{
		".",
		"├── projects/",
		"│   └── kessler.md",
		"├── work/",
		"│   ├── acme.md",
		"│   └── beta.md",
		"├── about.md",
		"└── contact.md",
		"",
		"2 directories, 5 files",
	}, "\n")
	if got, _ := run(t, s, "tree"); got != want {
		t.Fatalf("tree =\n%s\nwant\n%s", got, want)
	}
	if got, _ := run(t, s, "tree work"); !strings.HasPrefix(got, "work\n├── acme.md") {
		t.Fatalf("tree work =\n%s", got)
	}
}

func TestNotFoundAndSuggestions(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"sl":     "sl: command not found\ndid you mean ls?",
		"pdw":    "pdw: command not found\ndid you mean pwd?",
		"x":      "x: command not found",
		"banana": "banana: command not found",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 127 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestParsing(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "ls | grep x"); got != "pipes and redirects aren't supported here" || r.Code != 1 {
		t.Errorf("pipe = %q (%d)", got, r.Code)
	}
	if got, _ := run(t, s, "ls > out"); got != "pipes and redirects aren't supported here" {
		t.Errorf("redirect = %q", got)
	}
	if got, _ := run(t, s, `ls "work`); got != "unexpected EOF while looking for matching `\"'" {
		t.Errorf("unterminated = %q", got)
	}
	if got, _ := run(t, s, `ls "work"`); got != "acme.md  beta.md" {
		t.Errorf("quoted = %q", got)
	}
	if r := s.Run("   "); r.Output != "" || r.Code != 0 {
		t.Errorf("blank line = %+v", r)
	}
}

func TestHistoryRecordsNonBlankLines(t *testing.T) {
	s := newTestSession(t)
	s.Run("ls")
	s.Run("")
	s.Run("nope")
	if h := s.History(); len(h) != 2 || h[0] != "ls" || h[1] != "nope" {
		t.Fatalf("history = %q", h)
	}
}

// Review focus 2: control characters never reach the output.
func TestControlCharactersAreStripped(t *testing.T) {
	s := newTestSession(t)
	r := s.Run("ls \x1b[2J\x1b]0;pwned\x07work")
	if strings.Contains(r.Output, "\x1b[2J") || strings.Contains(r.Output, "\x1b]0;") {
		t.Fatalf("escape sequence leaked: %q", r.Output)
	}
	r = s.Run("\x1b[31mfoo")
	if strings.Contains(r.Output, "\x1b[31m") {
		t.Fatalf("escape sequence leaked: %q", r.Output)
	}
}

// Review focus 4: nothing outside the virtual tree is reachable.
func TestPathEscapes(t *testing.T) {
	s := newTestSession(t)
	for _, p := range []string{"../../../../etc/passwd", "/etc", "/home/guest/../../root", "~/../../proc"} {
		if _, r := run(t, s, "ls "+p); r.Code != 1 {
			t.Errorf("ls %s succeeded", p)
		}
	}
}
