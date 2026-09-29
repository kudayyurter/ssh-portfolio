package ui

import (
	"fmt"
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
	printed []string // text added to the scrollback by each message, in order
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
	sh := shell.New(fsys, content.Profile{}, shell.Options{Width: o.Width, Interactive: true})
	h.m = New(sh, o)
	h.record(nil)
	h.runCmd(h.m.Init())
	return h
}

// record notes whatever the last message added to the scrollback.
func (h *harness) record(before []string) {
	after := h.m.lines
	if len(after) < len(before) { // cleared or trimmed: everything is new
		before = nil
	}
	if added := after[len(before):]; len(added) > 0 {
		h.printed = append(h.printed, ansi.Strip(strings.Join(added, "\n")))
	}
}

// send runs one message through Update, then the command it returns: tea.Quit
// is noted and the model's own follow-up messages are fed back in.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	before := append([]string(nil), h.m.lines...)
	next, cmd := h.m.Update(msg)
	h.m = next.(Model)
	h.record(before)
	h.runCmd(cmd)
}

func (h *harness) runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch out := cmd().(type) {
	case tea.QuitMsg:
		h.quit = true
	case quitMsg:
		h.send(out)
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

func (h *harness) run(line string) {
	h.typeText(line)
	h.press(tea.KeyEnter)
}

func (h *harness) last() string {
	if len(h.printed) == 0 {
		return ""
	}
	return h.printed[len(h.printed)-1]
}

// screen is what the visitor sees: the view's rows without styling.
func (h *harness) screen() []string {
	return strings.Split(ansi.Strip(h.m.View().Content), "\n")
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
	h.send(tickMsg(t0.Add(boot.Duration)))
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("mode %v, printed %q", h.m.mode, h.printed)
	}
}

// The shell lives in the app's own full screen, not in the visitor's terminal.
func TestShellIsFullScreen(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	if !h.m.View().AltScreen {
		t.Fatal("boot should use the alt screen")
	}
	h.typeText("x")
	v := h.m.View()
	if !v.AltScreen {
		t.Fatal("the shell should stay in the alt screen")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode = %v, want cell motion for wheel scrolling", v.MouseMode)
	}
}

func TestOutputFlowsFromTheTop(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	want := []string{"welcome!", "guest@kuday:~$ "}
	if got := h.screen(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("screen =\n%q\nwant\n%q", got, want)
	}
	v := h.m.View()
	if v.Cursor == nil || v.Cursor.Y != 1 || v.Cursor.X != len("guest@kuday:~$ ") {
		t.Fatalf("cursor = %+v, want on the prompt row", v.Cursor)
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
	h.run("ls")
	if got := h.last(); got != "guest@kuday:~$ ls\nwork/  about.md" {
		t.Fatalf("printed %q", got)
	}
	if len(h.m.input) != 0 {
		t.Fatal("input not reset")
	}
	if len(logged) != 1 || logged[0] != "ls" {
		t.Fatalf("logged %q", logged)
	}
	h.run("cd work")
	if got := h.last(); got != "guest@kuday:~$ cd work" {
		t.Fatalf("echo should use the old cwd: %q", got)
	}
	if s := h.screen(); s[len(s)-1] != "guest@kuday:~/work$ " {
		t.Fatalf("prompt row = %q", s[len(s)-1])
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
	h.run("pwd")
	h.run("ls")
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

func TestCtrlDExits(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.press('d', tea.ModCtrl)
	if !h.quit || !strings.HasSuffix(h.last(), "exit\nlogout") {
		t.Fatalf("ctrl+d: quit %v printed %q", h.quit, h.last())
	}
}

func TestClearEmptiesTheScreen(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	h.run("ls")
	h.run("clear")
	if got := h.screen(); len(got) != 1 || got[0] != "guest@kuday:~$ " {
		t.Fatalf("after clear: %q", got)
	}
	h.run("ls")
	h.press('l', tea.ModCtrl)
	if got := h.screen(); len(got) != 1 {
		t.Fatalf("after ctrl+l: %q", got)
	}
}

func TestBootCommandReplays(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.typeText("x") // skip first boot
	h.run("boot")
	if h.m.mode != modeBoot {
		t.Fatal("boot should replay the animation")
	}
	h.typeText("q")
	if h.m.mode != modeShell || h.last() != "guest@kuday:~$ boot" {
		t.Fatalf("second skip: mode %v printed %q", h.m.mode, h.printed)
	}

	small := newHarness(t, Options{Width: 30, Height: 10})
	small.run("boot")
	if small.m.mode != modeShell || !strings.Contains(strings.ReplaceAll(small.last(), "\n", ""), "can't show the animation") {
		t.Fatalf("small boot: %q", small.last())
	}
}

// The notice must stay on screen long enough to read before the app closes.
func TestShutdownShowsNoticeThenQuits(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	next, cmd := h.m.Update(ShutdownMsg{})
	h.m = next.(Model)
	if !strings.Contains(strings.Join(h.screen(), "\n"), "system going down") {
		t.Fatalf("notice not on screen:\n%s", strings.Join(h.screen(), "\n"))
	}
	if cmd == nil {
		t.Fatal("no quit scheduled")
	}
	if _, ok := cmd().(quitMsg); !ok {
		t.Fatal("shutdown should quit after a delay, not at once")
	}
	next, cmd = h.m.Update(quitMsg{})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("quitMsg should quit")
	}
}

// fill prints n numbered lines through echo.
func fill(h *harness, n int) {
	for i := range n {
		h.run(fmt.Sprintf("echo line%02d", i))
	}
}

func TestScrollbackKeepsThePromptAtTheBottom(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	fill(h, 20) // 40 rows of output
	s := h.screen()
	if len(s) != 10 || s[9] != "guest@kuday:~$ " || s[8] != "line19" {
		t.Fatalf("bottom of screen =\n%s", strings.Join(s, "\n"))
	}
	if c := h.m.View().Cursor; c == nil || c.Y != 9 {
		t.Fatalf("cursor = %+v, want on the last row", c)
	}
}

func TestPageUpAndDown(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	fill(h, 20)
	h.press(tea.KeyPgUp)
	s := h.screen()
	if !strings.Contains(s[9], "scrolled up") || strings.Contains(strings.Join(s, "\n"), "line19") {
		t.Fatalf("after pgup:\n%s", strings.Join(s, "\n"))
	}
	if h.m.View().Cursor != nil {
		t.Fatal("cursor should be hidden while scrolled up")
	}
	for range 20 {
		h.press(tea.KeyPgUp)
	}
	if s := h.screen(); s[0] != "welcome!" {
		t.Fatalf("pgup should stop at the top, first row %q", s[0])
	}
	for range 20 {
		h.press(tea.KeyPgDown)
	}
	if s := h.screen(); s[9] != "guest@kuday:~$ " {
		t.Fatalf("pgdown should return to the prompt, last row %q", s[9])
	}
}

func TestShiftArrowsAndWheelScroll(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	fill(h, 20)
	h.press(tea.KeyUp, tea.ModShift)
	if h.m.scroll != 1 {
		t.Fatalf("shift+up scroll = %d", h.m.scroll)
	}
	h.press(tea.KeyDown, tea.ModShift)
	h.press(tea.KeyDown, tea.ModShift)
	if h.m.scroll != 0 {
		t.Fatalf("shift+down scroll = %d", h.m.scroll)
	}
	h.send(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if h.m.scroll != 3 {
		t.Fatalf("wheel up scroll = %d", h.m.scroll)
	}
	h.send(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if h.m.scroll != 0 {
		t.Fatalf("wheel down scroll = %d", h.m.scroll)
	}
	if string(h.m.input) != "" {
		t.Fatalf("scrolling typed into the prompt: %q", string(h.m.input))
	}
}

func TestTypingReturnsToThePrompt(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	fill(h, 20)
	h.press(tea.KeyPgUp)
	h.typeText("l")
	if s := h.screen(); h.m.scroll != 0 || s[9] != "guest@kuday:~$ l" {
		t.Fatalf("typing while scrolled: scroll %d, last row %q", h.m.scroll, s[9])
	}
}

func TestScrollbackIsCapped(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 10, Plain: true})
	h.m.print(strings.Repeat("x\n", 3000))
	if n := len(h.m.lines); n > maxLines {
		t.Fatalf("scrollback holds %d lines, cap %d", n, maxLines)
	}
}

func TestOutputRewrapsOnResize(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 20, Plain: true})
	h.run("echo " + strings.Repeat("a", 100))
	rows80 := len(h.screen())
	h.send(tea.WindowSizeMsg{Width: 40, Height: 20})
	rows40 := len(h.screen())
	if rows40 <= rows80 {
		t.Fatalf("narrower window should wrap into more rows: %d at 80, %d at 40", rows80, rows40)
	}
	for _, l := range h.screen() {
		if ansi.StringWidth(l) > 40 {
			t.Fatalf("row wider than 40: %q", l)
		}
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

// Review focus 3: tiny or unknown sizes wrap instead of panicking.
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
	h := newHarness(t, Options{Width: 80, Height: 0, Plain: true})
	h.send(tea.WindowSizeMsg{Width: 80, Height: 0})
	_ = h.m.View() // unknown height must not panic
}
