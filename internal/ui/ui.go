// Package ui is the Bubble Tea program each visitor runs. Everything happens
// in the app's own full screen: first the boot animation, then a shell with
// its own scrollback, so the visitor's terminal is untouched until they leave.
package ui

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
)

// ShutdownMsg tells a session the server is going down.
type ShutdownMsg struct{}

type tickMsg time.Time

// quitMsg ends the program after the shutdown notice has been on screen.
type quitMsg struct{}

type mode int

const (
	modeBoot mode = iota
	modeShell
)

const (
	maxInput     = 1024                    // runes; longer pastes are cut
	maxLines     = 2000                    // scrollback lines kept per session
	wheelStep    = 3                       // rows per mouse-wheel notch
	shutdownWait = 1500 * time.Millisecond // how long the shutdown notice shows
)

// Options configure a Model.
type Options struct {
	Width, Height int
	Plain         bool              // no animation (colorless or dumb terminal)
	Boot          boot.Options      // words under the name
	Welcome       string            // shown when the shell starts
	OnCommand     func(line string) // called for every non-blank line (logging)
}

// Model is one visitor's screen.
type Model struct {
	sh       *shell.Session
	o        Options
	mode     mode
	w, h     int
	start    time.Time // first tick of the current boot run
	now      time.Time // latest tick
	welcomed bool
	lines    []string // scrollback, oldest first; each may be wider than the window
	scroll   int      // rows scrolled up from the prompt; 0 follows the prompt
	input    []rune
	cursor   int
	hist     int    // history index while browsing; len(history) when on a fresh line
	draft    []rune // the fresh line saved while browsing history
	tabbed   bool   // previous key was a Tab that had several candidates
	used     bool   // a command has run, so the hint is hidden
}

// New decides whether to animate: plain terminals and windows too small for
// the animation start straight in the shell.
func New(sh *shell.Session, o Options) Model {
	m := Model{sh: sh, o: o, w: o.Width, h: o.Height}
	if o.Plain || !boot.Fits(o.Width, o.Height) {
		m.mode = modeShell
		m.welcome()
	}
	return m
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init starts the animation when there is one.
func (m Model) Init() tea.Cmd {
	if m.mode == modeBoot {
		return tick()
	}
	return nil
}

// Update handles one message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.sh.SetWidth(msg.Width)
		m.scroll = min(m.scroll, m.maxScroll())
		return m, nil

	case ShutdownMsg:
		m.enterShell()
		m.print(style.Muted.Render("system going down for update, reconnect in a moment"))
		return m, tea.Tick(shutdownWait, func(time.Time) tea.Msg { return quitMsg{} })

	case quitMsg:
		return m, tea.Quit

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
			m.enterShell()
			return m, nil
		}
		return m, tick()

	case tea.PasteMsg:
		if m.mode == modeBoot {
			m.enterShell() // swallow: a paste only skips the animation
			return m, nil
		}
		m.scroll = 0
		m.insert(clean(msg.Content))
		return m, nil

	case tea.MouseWheelMsg:
		if m.mode == modeShell {
			switch msg.Mouse().Button {
			case tea.MouseWheelUp:
				m.scrollBy(wheelStep)
			case tea.MouseWheelDown:
				m.scrollBy(-wheelStep)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeBoot {
			m.enterShell() // swallow: any key only skips the animation
			return m, nil
		}
		return m.key(msg)
	}
	return m, nil
}

func (m *Model) enterShell() {
	m.mode = modeShell
	m.start, m.now = time.Time{}, time.Time{}
	m.welcome()
}

func (m *Model) welcome() {
	if !m.welcomed {
		m.welcomed = true
		m.print(m.o.Welcome)
	}
}

// print adds output to the scrollback and jumps back to the prompt.
func (m *Model) print(s string) {
	m.lines = append(m.lines, strings.Split(s, "\n")...)
	if over := len(m.lines) - maxLines; over > 0 {
		m.lines = slices.Delete(m.lines, 0, over)
	}
	m.scroll = 0
}

func (m *Model) scrollBy(rows int) {
	m.scroll = max(0, min(m.scroll+rows, m.maxScroll()))
}

func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	page := max(1, m.height()-1)
	switch k.String() {
	case "pgup":
		m.scrollBy(page)
		return m, nil
	case "pgdown":
		m.scrollBy(-page)
		return m, nil
	case "shift+up":
		m.scrollBy(1)
		return m, nil
	case "shift+down":
		m.scrollBy(-1)
		return m, nil
	}

	m.scroll = 0 // any other key returns to the prompt
	wasTab := m.tabbed
	m.tabbed = false
	switch k.String() {
	case "enter":
		return m.submit()
	case "ctrl+c":
		m.print(m.promptLine() + "^C")
		m.resetLine()
	case "ctrl+d":
		if len(m.input) == 0 {
			m.input = []rune("exit")
			return m.submit()
		}
	case "ctrl+l":
		m.lines = nil
	case "tab":
		m.complete(wasTab)
	case "backspace", "ctrl+h":
		if m.cursor > 0 {
			m.input = slices.Delete(m.input, m.cursor-1, m.cursor)
			m.cursor--
		}
	case "delete":
		if m.cursor < len(m.input) {
			m.input = slices.Delete(m.input, m.cursor, m.cursor+1)
		}
	case "left", "ctrl+b":
		m.cursor = max(0, m.cursor-1)
	case "right", "ctrl+f":
		m.cursor = min(len(m.input), m.cursor+1)
	case "home", "ctrl+a":
		m.cursor = 0
	case "end", "ctrl+e":
		m.cursor = len(m.input)
	case "ctrl+u":
		m.input = slices.Clone(m.input[m.cursor:])
		m.cursor = 0
	case "ctrl+w":
		i := m.cursor
		for i > 0 && m.input[i-1] == ' ' {
			i--
		}
		for i > 0 && m.input[i-1] != ' ' {
			i--
		}
		m.input = slices.Delete(m.input, i, m.cursor)
		m.cursor = i
	case "up":
		m.historyPrev()
	case "down":
		m.historyNext()
	default:
		if t := k.Key().Text; t != "" {
			m.insert(clean(t))
		}
	}
	return m, nil
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	line := string(m.input)
	echo := m.promptLine() // uses the directory the command was typed in
	m.resetLine()
	if strings.TrimSpace(line) != "" {
		m.used = true
		if m.o.OnCommand != nil {
			m.o.OnCommand(line)
		}
	}
	res := m.sh.Run(line)
	m.hist = len(m.sh.History())

	switch res.Action {
	case shell.ActionClear:
		m.lines = nil
		return m, nil
	case shell.ActionExit:
		m.print(echo + "\n" + res.Output)
		return m, tea.Quit
	case shell.ActionBoot:
		if m.o.Plain || !boot.Fits(m.w, m.h) {
			m.print(echo + "\n" + "boot: this window can't show the animation (too small or no color)")
			return m, nil
		}
		m.print(echo)
		m.mode = modeBoot
		return m, tick()
	}
	if res.Output != "" {
		echo += "\n" + res.Output
	}
	m.print(echo)
	return m, nil
}

func (m *Model) complete(wasTab bool) {
	before, after := string(m.input[:m.cursor]), m.input[m.cursor:]
	got, cands := m.sh.Complete(before)
	if got != before {
		m.input = append([]rune(got), after...)
		m.cursor = len([]rune(got))
		return
	}
	if len(cands) < 2 {
		return
	}
	m.tabbed = true
	if wasTab { // like bash: the second Tab lists the candidates
		m.print(m.promptLine() + "\n" + strings.Join(cands, "  "))
	}
}

func (m *Model) insert(s string) {
	r := []rune(s)
	if room := maxInput - len(m.input); len(r) > room {
		r = r[:max(0, room)]
	}
	m.input = slices.Insert(m.input, m.cursor, r...)
	m.cursor += len(r)
}

func (m *Model) resetLine() {
	m.input, m.cursor, m.draft = nil, 0, nil
	m.hist = len(m.sh.History())
}

func (m *Model) setInput(r []rune) {
	m.input = slices.Clone(r)
	m.cursor = len(m.input)
}

func (m *Model) historyPrev() {
	h := m.sh.History()
	if m.hist > len(h) {
		m.hist = len(h)
	}
	if m.hist == 0 {
		return
	}
	if m.hist == len(h) {
		m.draft = slices.Clone(m.input)
	}
	m.hist--
	m.setInput([]rune(h[m.hist]))
}

func (m *Model) historyNext() {
	h := m.sh.History()
	if m.hist >= len(h) {
		return
	}
	m.hist++
	if m.hist == len(h) {
		m.setInput(m.draft)
	} else {
		m.setInput([]rune(h[m.hist]))
	}
}

// clean makes pasted or typed text safe for a single command line: line
// breaks and tabs become spaces, other control characters are dropped.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, s)
}

func (m Model) promptLine() string { return m.sh.Prompt() + string(m.input) }

func (m Model) width() int {
	if m.w <= 0 {
		return 80
	}
	return m.w
}

func (m Model) height() int {
	if m.h <= 0 {
		return 24
	}
	return m.h
}

// rows lays out the whole shell screen at the current width: scrollback,
// the hint (until the first command) and the prompt. It also returns where
// the cursor sits, counted in rows from the top of the layout.
func (m Model) rows() (rows []string, cursorRow, cursorCol int) {
	w := m.width()
	for _, l := range m.lines {
		rows = append(rows, strings.Split(ansi.Hardwrap(l, w, true), "\n")...)
	}
	if !m.used {
		rows = append(rows, style.Faint.Render(ansi.Truncate("try: ls · cat about.md · help", w, "")))
	}
	pos := ansi.StringWidth(m.sh.Prompt()) + ansi.StringWidth(string(m.input[:m.cursor]))
	cursorRow, cursorCol = len(rows)+pos/w, pos%w
	rows = append(rows, strings.Split(ansi.Hardwrap(m.promptLine(), w, true), "\n")...)
	return rows, cursorRow, cursorCol
}

func (m Model) maxScroll() int {
	rows, _, _ := m.rows()
	return max(0, len(rows)-m.height())
}

// View is the animation in boot mode, otherwise the window's worth of shell
// rows ending m.scroll rows above the prompt.
func (m Model) View() tea.View {
	if m.mode == modeBoot {
		v := tea.NewView(boot.Frame(m.now.Sub(m.start), m.w, m.h, m.o.Boot))
		v.AltScreen = true
		return v
	}
	rows, cursorRow, cursorCol := m.rows()
	h := m.height()
	scroll := min(m.scroll, max(0, len(rows)-h))
	end := len(rows) - scroll
	top := max(0, end-h)
	visible := slices.Clone(rows[top:end])
	if scroll > 0 {
		visible[len(visible)-1] = style.Faint.Render(ansi.Truncate("── scrolled up · PgDn or type to return ──", m.width(), ""))
	}

	v := tea.NewView(strings.Join(visible, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if scroll == 0 {
		v.Cursor = tea.NewCursor(cursorCol, cursorRow-top)
	}
	return v
}
