// Package ui is the Bubble Tea program each visitor runs. Everything happens
// in the app's own full screen: first the boot animation, then the portfolio
// TUI, so the visitor's terminal is untouched until they leave.
package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/tui"
)

// ShutdownMsg tells a session the server is going down.
type ShutdownMsg struct{}

type tickMsg time.Time

// quitMsg ends the program after the shutdown notice has been on screen.
type quitMsg struct{}

type mode int

const (
	modeBoot mode = iota
	modeTUI
)

const shutdownWait = 1500 * time.Millisecond // how long the shutdown notice shows

const shutdownNotice = "system going down for update, reconnect in a moment"

// Alternate scroll mode: in the alt screen the terminal turns mouse-wheel
// motion into ↑/↓ keys. The wheel scrolls pages without the program
// capturing the mouse, so the terminal still handles clicks on links.
const (
	altScrollOn  = "\x1b[?1007h"
	altScrollOff = "\x1b[?1007l"
)

// Options configure a Model.
type Options struct {
	Width, Height int
	Plain         bool              // no animation (colorless or dumb terminal)
	Boot          boot.Options      // words under the name
	OnOpen        func(path string) // called for every page opened (logging)
}

// Model is one visitor's screen.
type Model struct {
	tui    tui.Model
	o      Options
	mode   mode
	w, h   int
	start  time.Time // first tick of the boot animation
	now    time.Time // latest tick
	notice string    // shown on the bottom row, e.g. the shutdown warning
}

// New decides whether to animate: plain terminals and windows too small for
// the animation start straight in the TUI.
func New(t tui.Model, o Options) Model {
	m := Model{tui: t, o: o, w: o.Width, h: o.Height}
	if o.Plain || !boot.Fits(o.Width, o.Height) {
		m.mode = modeTUI
	}
	return m
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// quit resets alternate scroll before the program ends.
func quit() tea.Cmd { return tea.Sequence(tea.Raw(altScrollOff), tea.Quit) }

// Init starts the animation, or turns on alternate scroll for the TUI.
func (m Model) Init() tea.Cmd {
	if m.mode == modeBoot {
		return tick()
	}
	return tea.Raw(altScrollOn)
}

// enterTUI ends the animation if it is running and turns on alternate scroll.
func (m *Model) enterTUI() tea.Cmd {
	if m.mode == modeTUI {
		return nil
	}
	m.mode = modeTUI
	m.start, m.now = time.Time{}, time.Time{}
	return tea.Raw(altScrollOn)
}

// Update handles one message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.tui.SetSize(msg.Width, msg.Height)
		return m, nil

	case ShutdownMsg:
		cmd := m.enterTUI()
		m.notice = shutdownNotice
		return m, tea.Batch(cmd, tea.Tick(shutdownWait, func(time.Time) tea.Msg { return quitMsg{} }))

	case quitMsg:
		return m, quit()

	case tickMsg:
		if m.mode != modeBoot {
			return m, nil
		}
		now := time.Time(msg)
		if m.start.IsZero() {
			m.start = now
		}
		m.now = now
		if now.Sub(m.start) >= boot.Duration {
			return m, m.enterTUI()
		}
		return m, tick()

	case tea.PasteMsg:
		if m.mode == modeBoot {
			return m, m.enterTUI() // swallow: a paste only skips the animation
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeBoot {
			return m, m.enterTUI() // swallow: any key only skips the animation
		}
		var res tui.Result
		m.tui, res = m.tui.Update(msg)
		if res.Opened != "" && m.o.OnOpen != nil {
			m.o.OnOpen(res.Opened)
		}
		if res.Quit {
			return m, quit()
		}
		return m, nil
	}
	return m, nil
}

// View is the animation in boot mode, otherwise the TUI with any notice on
// its bottom row. The mouse is never captured and the cursor stays hidden.
func (m Model) View() tea.View {
	var content string
	if m.mode == modeBoot {
		content = boot.Frame(m.now.Sub(m.start), m.w, m.h, m.o.Boot)
	} else {
		content = m.tui.View()
		if m.notice != "" {
			w := m.w
			if w <= 0 {
				w = 80
			}
			rows := strings.Split(content, "\n")
			rows[len(rows)-1] = style.Muted.Render(ansi.Truncate(m.notice, w, ""))
			content = strings.Join(rows, "\n")
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
