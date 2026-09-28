package ui

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

type harness struct {
	t       *testing.T
	m       Model
	printed []string
	quit    bool
}

func newHarness(t *testing.T, o Options) *harness {
	t.Helper()
	fsys, err := vfs.New(fstest.MapFS{
		"about.md":     {Data: []byte("---\ntitle: About\nsummary: Who\n---\nHi.\n")},
		"work/acme.md": {Data: []byte("---\ntitle: Acme\nsummary: Job\n---\nWork.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t}
	o.Welcome = "welcome!"
	o.Print = func(s string) tea.Cmd {
		h.printed = append(h.printed, ansi.Strip(s))
		return nil
	}
	sh := shell.New(fsys, content.Profile{}, shell.Options{Width: o.Width, Interactive: true})
	h.m = New(sh, o)
	h.m.Init()
	return h
}

// send runs one message through Update and executes the returned command
// to notice tea.Quit. The recording Print returns nil, so tea.Sequence
// collapses to the single remaining command.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	next, cmd := h.m.Update(msg)
	h.m = next.(Model)
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			h.quit = true
		}
	}
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		h.send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func (h *harness) press(code rune, mod ...tea.KeyMod) {
	k := tea.KeyPressMsg{Code: code}
	for _, m := range mod {
		k.Mod |= m
	}
	h.send(k)
}

func (h *harness) last() string {
	if len(h.printed) == 0 {
		return ""
	}
	return h.printed[len(h.printed)-1]
}

func TestSmallWindowSkipsBoot(t *testing.T) {
	h := newHarness(t, Options{Width: 30, Height: 10})
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("30×10 should start in the shell with the welcome: mode %v printed %q", h.m.mode, h.printed)
	}
}

func TestPlainSkipsBootAndWelcomes(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30, Plain: true})
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("plain terminal should start in the shell: mode %v printed %q", h.m.mode, h.printed)
	}
}

func TestBootEndsAfterDuration(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	if h.m.mode != modeBoot {
		t.Fatal("should start in boot mode")
	}
	t0 := time.Unix(0, 0)
	h.send(tickMsg(t0))
	h.send(tickMsg(t0.Add(time.Second)))
	if h.m.mode != modeBoot {
		t.Fatal("left boot too early")
	}
	if !h.m.View().AltScreen {
		t.Fatal("boot should use the alt screen")
	}
	h.send(tickMsg(t0.Add(boot.Duration)))
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("mode %v, printed %q", h.m.mode, h.printed)
	}
	if h.m.View().AltScreen {
		t.Fatal("shell should be inline")
	}
}

// Review focus 5: input during boot skips and is swallowed.
func TestKeysAndPastesDuringBootAreSwallowed(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.typeText("x")
	if h.m.mode != modeShell || len(h.m.input) != 0 {
		t.Fatalf("key during boot: mode %v input %q", h.m.mode, string(h.m.input))
	}
	h = newHarness(t, Options{Width: 100, Height: 30})
	h.send(tea.PasteMsg{Content: "rm -rf /"})
	if h.m.mode != modeShell || len(h.m.input) != 0 {
		t.Fatalf("paste during boot: mode %v input %q", h.m.mode, string(h.m.input))
	}
}

func TestRunCommand(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	var logged []string
	h.m.o.OnCommand = func(l string) { logged = append(logged, l) }
	if !strings.Contains(ansi.Strip(h.m.View().Content), "try: ls") {
		t.Fatal("hint missing before the first command")
	}
	h.typeText("ls")
	h.press(tea.KeyEnter)
	if got := h.last(); got != "guest@kuday:~$ ls\nwork/  about.md" {
		t.Fatalf("printed %q", got)
	}
	if len(h.m.input) != 0 || strings.Contains(ansi.Strip(h.m.View().Content), "try: ls") {
		t.Fatal("input not reset or hint still shown")
	}
	if len(logged) != 1 || logged[0] != "ls" {
		t.Fatalf("logged %q", logged)
	}
	h.typeText("cd work")
	h.press(tea.KeyEnter)
	if got := h.last(); got != "guest@kuday:~$ cd work" {
		t.Fatalf("echo should use the old cwd: %q", got)
	}
	if got := ansi.Strip(h.m.View().Content); got != "guest@kuday:~/work$ " {
		t.Fatalf("view = %q", got)
	}
}

func TestLineEditing(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.typeText("cat abut")
	h.press(tea.KeyLeft)
	h.press(tea.KeyLeft)
	h.typeText("o")
	if string(h.m.input) != "cat about" {
		t.Fatalf("insert at cursor: %q", string(h.m.input))
	}
	h.press(tea.KeyBackspace)
	h.press(tea.KeyHome)
	h.press(tea.KeyDelete)
	if string(h.m.input) != "at abut" {
		t.Fatalf("backspace/delete: %q", string(h.m.input))
	}
	h.press(tea.KeyEnd)
	h.press('w', tea.ModCtrl)
	if string(h.m.input) != "at " {
		t.Fatalf("ctrl+w: %q", string(h.m.input))
	}
	h.press('u', tea.ModCtrl)
	if string(h.m.input) != "" {
		t.Fatalf("ctrl+u: %q", string(h.m.input))
	}
	h.typeText("pwd")
	h.press('c', tea.ModCtrl)
	if len(h.m.input) != 0 || h.last() != "guest@kuday:~$ pwd^C" {
		t.Fatalf("ctrl+c: input %q printed %q", string(h.m.input), h.last())
	}
}

func TestHistory(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	for _, l := range []string{"pwd", "ls"} {
		h.typeText(l)
		h.press(tea.KeyEnter)
	}
	h.typeText("dra")
	h.press(tea.KeyUp)
	if string(h.m.input) != "ls" {
		t.Fatalf("up: %q", string(h.m.input))
	}
	h.press(tea.KeyUp)
	h.press(tea.KeyUp) // stays at the oldest
	if string(h.m.input) != "pwd" {
		t.Fatalf("up up: %q", string(h.m.input))
	}
	h.press(tea.KeyDown)
	h.press(tea.KeyDown)
	if string(h.m.input) != "dra" {
		t.Fatalf("down restores draft: %q", string(h.m.input))
	}
}

func TestTabCompletion(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.typeText("ca")
	h.press(tea.KeyTab)
	if string(h.m.input) != "cat " {
		t.Fatalf("tab: %q", string(h.m.input))
	}
	h.press(tea.KeyTab)
	n := len(h.printed)
	h.press(tea.KeyTab)
	if len(h.printed) != n+1 || !strings.HasSuffix(h.last(), "work/  about.md") {
		t.Fatalf("second tab should list candidates: %q", h.printed)
	}
}

func TestClearAndExit(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.press('d', tea.ModCtrl)
	if !h.quit || !strings.HasSuffix(h.last(), "exit\nlogout") {
		t.Fatalf("ctrl+d: quit %v printed %q", h.quit, h.last())
	}
}

func TestBootCommandReplays(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.typeText("x") // skip first boot
	h.typeText("boot")
	h.press(tea.KeyEnter)
	if h.m.mode != modeBoot {
		t.Fatal("boot should replay the animation")
	}
	h.typeText("q")
	if h.m.mode != modeShell || h.last() != "guest@kuday:~$ boot" {
		t.Fatalf("second skip: mode %v printed %q", h.m.mode, h.printed)
	}

	small := newHarness(t, Options{Width: 30, Height: 10})
	small.typeText("boot")
	small.press(tea.KeyEnter)
	if small.m.mode != modeShell || !strings.Contains(strings.ReplaceAll(small.last(), "\n", ""), "can't show the animation") {
		t.Fatalf("small boot: %q", small.last())
	}
}

func TestShutdownPrintsAndQuits(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.send(ShutdownMsg{})
	if !h.quit || !strings.Contains(h.last(), "system going down") {
		t.Fatalf("shutdown: quit %v printed %q", h.quit, h.printed)
	}
}

// Review focus 2: pasted newlines and escapes never reach the command line.
func TestPasteIsCleaned(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.send(tea.PasteMsg{Content: "echo a\nb\x1b[31m\tc"})
	if got := string(h.m.input); got != "echo a b[31m c" {
		t.Fatalf("pasted = %q", got)
	}
	h.send(tea.PasteMsg{Content: strings.Repeat("x", 5000)})
	if len(h.m.input) != maxInput {
		t.Fatalf("input length %d, want cap %d", len(h.m.input), maxInput)
	}
}

// Review focus 3: tiny or unknown widths wrap instead of panicking.
func TestViewWrapsInNarrowWindows(t *testing.T) {
	for _, w := range []int{0, 1, 10} {
		h := newHarness(t, Options{Width: w, Height: 24, Plain: true})
		h.send(tea.WindowSizeMsg{Width: w, Height: 24})
		h.typeText(strings.Repeat("a", 300))
		v := h.m.View()
		limit := w
		if w <= 0 {
			limit = 80
		}
		for _, l := range strings.Split(ansi.Strip(v.Content), "\n") {
			if ansi.StringWidth(l) > limit {
				t.Fatalf("width %d: line of %d columns", w, ansi.StringWidth(l))
			}
		}
		if v.Cursor == nil || v.Cursor.X >= limit {
			t.Fatalf("width %d: bad cursor %+v", w, v.Cursor)
		}
	}
}

func TestTallOutputIsPrintedInPiecesThatFit(t *testing.T) {
	h := newHarness(t, Options{Width: 40, Height: 10, Plain: true})
	h.typeText("echo " + strings.Repeat("word ", 60)) // one 300-column line: 8 rows at width 40
	h.press(tea.KeyEnter)
	h.typeText("history")
	h.press(tea.KeyEnter)
	for _, p := range h.printed {
		rows := 0
		for _, l := range strings.Split(p, "\n") {
			rows += 1 + max(0, ansi.StringWidth(l)-1)/40
		}
		if rows > 7 {
			t.Fatalf("printed piece of %d rows in a 10-row window:\n%s", rows, p)
		}
	}
	all := strings.ReplaceAll(strings.Join(h.printed, ""), "\n", "") // undo wrapping
	if !strings.Contains(all, "1  echo word") || !strings.Contains(all, "2  history") || strings.Count(all, "word") != 180 {
		t.Fatalf("output lost when split:\n%s", all)
	}
}
