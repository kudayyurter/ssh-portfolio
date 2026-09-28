// Package ui is the Bubble Tea program each visitor runs: the boot animation
// in the alternate screen, then an inline shell whose output is printed into
// the terminal's own scrollback (so scrolling and copying work natively).
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

// printLaterMsg prints text (and optionally quits) once the alt screen is
// gone: text printed while the alt screen is up is thrown away with it.
type printLaterMsg struct {
	text string
	quit bool
}

// startBootMsg enters the animation after the command echo has printed.
type startBootMsg struct{}

// later sends printLaterMsg two frames from now, after the renderer has
// switched back to the normal screen.
func later(text string, quit bool) tea.Cmd {
	return tea.Tick(2*time.Second/30, func(time.Time) tea.Msg { return printLaterMsg{text, quit} })
}

type mode int

const (
	modeBoot mode = iota
	modeShell
)

const maxInput = 1024 // runes; longer pastes are cut

// Options configure a Model.
type Options struct {
	Width, Height int
	Plain         bool                 // no animation (colorless or dumb terminal)
	Boot          boot.Options         // words under the name
	Welcome       string               // printed when the shell starts
	OnCommand     func(line string)    // called for every non-blank line (logging)
	Print         func(string) tea.Cmd // nil means tea.Println; tests record output
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
	if o.Print == nil {
		o.Print = func(s string) tea.Cmd { return tea.Println(s) }
	}
	m := Model{sh: sh, o: o, w: o.Width, h: o.Height}
	if o.Plain || !boot.Fits(o.Width, o.Height) {
		m.mode = modeShell
		m.welcomed = true // Init prints it
	}
	return m
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init starts the animation, or prints the welcome when there is none.
func (m Model) Init() tea.Cmd {
	if m.mode == modeBoot {
		return tick()
	}
	return m.print(m.o.Welcome)
}

// Update handles one message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.sh.SetWidth(msg.Width)
		return m, nil

	case ShutdownMsg:
		notice := style.Muted.Render("system going down for update, reconnect in a moment")
		if m.mode == modeBoot {
			m.mode = modeShell
			return m, later(notice, true)
		}
		return m, tea.Sequence(m.print(notice), tea.Quit)

	case printLaterMsg:
		if msg.quit {
			return m, tea.Sequence(m.print(msg.text), tea.Quit)
		}
		return m, m.print(msg.text)

	case startBootMsg:
		m.mode = modeBoot
		return m, tick()

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
			return m.enterShell()
		}
		return m, tick()

	case tea.PasteMsg:
		if m.mode == modeBoot {
			return m.enterShell() // swallow: a paste only skips the animation
		}
		m.insert(clean(msg.Content))
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeBoot {
			return m.enterShell() // swallow: any key only skips the animation
		}
		return m.key(msg)
	}
	return m, nil
}

func (m Model) enterShell() (tea.Model, tea.Cmd) {
	m.mode = modeShell
	m.start, m.now = time.Time{}, time.Time{}
	if m.welcomed {
		return m, nil
	}
	m.welcomed = true
	return m, later(m.o.Welcome, false)
}

func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	wasTab := m.tabbed
	m.tabbed = false
	switch k.String() {
	case "enter":
		return m.submit()
	case "ctrl+c":
		echo := m.promptLine() + "^C"
		m.resetLine()
		return m, m.print(echo)
	case "ctrl+d":
		if len(m.input) == 0 {
			m.input = []rune("exit")
			return m.submit()
		}
	case "ctrl+l":
		return m, tea.ClearScreen
	case "tab":
		return m.complete(wasTab)
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
		return m, tea.ClearScreen
	case shell.ActionExit:
		return m, tea.Sequence(m.print(echo+"\n"+res.Output), tea.Quit)
	case shell.ActionBoot:
		if m.o.Plain || !boot.Fits(m.w, m.h) {
			return m, m.print(echo + "\n" + "boot: this window can't show the animation (too small or no color)")
		}
		return m, tea.Sequence(m.print(echo), func() tea.Msg { return startBootMsg{} })
	}
	if res.Output != "" {
		echo += "\n" + res.Output
	}
	return m, m.print(echo)
}

func (m Model) complete(wasTab bool) (tea.Model, tea.Cmd) {
	before, after := string(m.input[:m.cursor]), m.input[m.cursor:]
	got, cands := m.sh.Complete(before)
	if got != before {
		m.input = append([]rune(got), after...)
		m.cursor = len([]rune(got))
		return m, nil
	}
	if len(cands) < 2 {
		return m, nil
	}
	m.tabbed = true
	if !wasTab {
		return m, nil // like bash: the second Tab lists the candidates
	}
	return m, m.print(m.promptLine() + "\n" + strings.Join(cands, "  "))
}

// print shows s above the prompt. Bubble Tea inserts printed text above the
// view in one go, and a piece taller than the window pushes the prompt off
// the screen, after which the renderer loses track of it. So output is
// hard-wrapped to the window and sent in pieces that each fit.
func (m Model) print(s string) tea.Cmd {
	room := m.h - 3 // leave the hint and prompt lines on screen
	if m.h <= 0 {
		room = 20
	}
	room = max(1, room)
	lines := strings.Split(ansi.Hardwrap(s, m.width(), true), "\n")
	var cmds []tea.Cmd
	for len(lines) > 0 {
		n := min(room, len(lines))
		cmds = append(cmds, m.o.Print(strings.Join(lines[:n], "\n")))
		lines = lines[n:]
	}
	return tea.Sequence(cmds...)
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

// View is the animation in boot mode, otherwise the hint (until the first
// command) and the prompt with the cursor, hard-wrapped to the window.
func (m Model) View() tea.View {
	if m.mode == modeBoot {
		v := tea.NewView(boot.Frame(m.now.Sub(m.start), m.w, m.h, m.o.Boot))
		v.AltScreen = true
		return v
	}
	w := m.width()
	var b strings.Builder
	top := 0
	if !m.used {
		b.WriteString(style.Faint.Render(ansi.Truncate("try: ls · cat about.md · help", w, "")) + "\n")
		top = 1
	}
	b.WriteString(ansi.Hardwrap(m.promptLine(), w, true))
	v := tea.NewView(b.String())
	pos := ansi.StringWidth(m.sh.Prompt()) + ansi.StringWidth(string(m.input[:m.cursor]))
	v.Cursor = tea.NewCursor(pos%w, top+pos/w)
	return v
}
