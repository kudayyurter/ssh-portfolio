package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/tui"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

type harness struct {
	t      *testing.T
	m      Model
	raw    []string // escape sequences sent with tea.Raw, in order
	opened []string // paths reported through OnOpen
	quit   bool
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
	o.OnOpen = func(p string) { h.opened = append(h.opened, p) }
	profile := content.Profile{Name: "Test Person", Tagline: "tester of things"}
	h.m = New(tui.New(fsys, profile, o.Width, o.Height), o)
	h.runCmd(h.m.Init())
	return h
}

// send runs one message through Update, then the command it returns.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	next, cmd := h.m.Update(msg)
	h.m = next.(Model)
	h.runCmd(cmd)
}

// runCmd runs a command the way Bubble Tea would, as far as the tests care:
// raw output is recorded, tea.Quit is noted, batches and sequences are
// unpacked in order, and the model's own follow-up messages are fed back.
// Animation ticks are dropped so boot tests control time themselves.
func (h *harness) runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch out := cmd().(type) {
	case tea.QuitMsg:
		h.quit = true
	case tea.RawMsg:
		h.raw = append(h.raw, fmt.Sprint(out.Msg))
	case quitMsg:
		h.send(out)
	case tea.BatchMsg:
		for _, c := range out {
			h.runCmd(c)
		}
	case tickMsg:
	default:
		// tea.Sequence's message type is unexported; it is a slice of commands.
		if v := reflect.ValueOf(out); v.Kind() == reflect.Slice {
			for i := range v.Len() {
				if c, ok := v.Index(i).Interface().(tea.Cmd); ok {
					h.runCmd(c)
				}
			}
		}
	}
}

func (h *harness) press(code rune, text string) {
	h.send(tea.KeyPressMsg{Code: code, Text: text})
}

// screen is what the visitor sees, without styling.
func (h *harness) screen() string { return ansi.Strip(h.m.View().Content) }

func TestSmallWindowSkipsBoot(t *testing.T) {
	h := newHarness(t, Options{Width: 30, Height: 10})
	if h.m.mode != modeTUI || !strings.Contains(h.screen(), "About") {
		t.Fatalf("30×10 should start at home: mode %v\n%s", h.m.mode, h.screen())
	}
	if len(h.raw) != 1 || h.raw[0] != altScrollOn {
		t.Fatalf("raw = %q, want alternate scroll on", h.raw)
	}
}

func TestPlainSkipsBoot(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30, Plain: true})
	if h.m.mode != modeTUI || !strings.Contains(h.screen(), "Test Person") {
		t.Fatalf("plain terminal should start at home: mode %v\n%s", h.m.mode, h.screen())
	}
}

func TestBootEndsAfterDuration(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	if h.m.mode != modeBoot || len(h.raw) != 0 {
		t.Fatalf("should start in boot mode without alternate scroll: mode %v raw %q", h.m.mode, h.raw)
	}
	t0 := time.Unix(0, 0)
	h.send(tickMsg(t0))
	h.send(tickMsg(t0.Add(time.Second)))
	if h.m.mode != modeBoot {
		t.Fatal("left boot too early")
	}
	h.send(tickMsg(t0.Add(boot.Duration)))
	if h.m.mode != modeTUI || len(h.raw) != 1 || h.raw[0] != altScrollOn {
		t.Fatalf("mode %v raw %q", h.m.mode, h.raw)
	}
}

// Input during boot only skips the animation.
func TestKeysAndPastesDuringBootAreSwallowed(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.press('j', "j")
	if h.m.mode != modeTUI || !strings.Contains(h.screen(), "▸ About") {
		t.Fatalf("key during boot: mode %v\n%s", h.m.mode, h.screen())
	}
	h = newHarness(t, Options{Width: 100, Height: 30})
	h.send(tea.PasteMsg{Content: "q"})
	if h.m.mode != modeTUI || h.quit {
		t.Fatalf("paste during boot: mode %v quit %v", h.m.mode, h.quit)
	}
}

// Full screen, no mouse capture (so links stay clickable), cursor hidden.
func TestFullScreenWithoutMouseCapture(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	v := h.m.View()
	if !v.AltScreen || v.MouseMode != tea.MouseModeNone || v.Cursor != nil {
		t.Fatalf("alt %v mouse %v cursor %+v", v.AltScreen, v.MouseMode, v.Cursor)
	}
}

func TestOpeningAPageIsReported(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.press('j', "j")
	h.send(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(h.opened) != 1 || h.opened[0] != vfs.Home+"/work" || !strings.Contains(h.screen(), "Acme") {
		t.Fatalf("opened %q\n%s", h.opened, h.screen())
	}
}

func TestQuitResetsAlternateScroll(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.press('q', "q")
	if !h.quit || h.raw[len(h.raw)-1] != altScrollOff {
		t.Fatalf("quit %v raw %q", h.quit, h.raw)
	}
}

func TestShutdownShowsNoticeThenQuits(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.send(ShutdownMsg{})
	rows := strings.Split(h.screen(), "\n")
	if !strings.Contains(rows[len(rows)-1], "system going down") || !h.quit || h.raw[len(h.raw)-1] != altScrollOff {
		t.Fatalf("last row %q quit %v raw %q", rows[len(rows)-1], h.quit, h.raw)
	}
}

// Review focus 4: shutdown during the animation still reaches the visitor.
func TestShutdownDuringBoot(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.send(ShutdownMsg{})
	if h.m.mode != modeTUI || !strings.Contains(h.screen(), "system going down") || !h.quit {
		t.Fatalf("mode %v quit %v\n%s", h.m.mode, h.quit, h.screen())
	}
}

func TestResizeReachesTheTUI(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.send(tea.WindowSizeMsg{Width: 30, Height: 7})
	if h.screen() != "make the window a bit bigger" {
		t.Fatalf("screen %q", h.screen())
	}
}
