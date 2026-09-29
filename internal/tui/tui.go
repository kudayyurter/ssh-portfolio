// Package tui is the portfolio as menus and pages: a home card, section
// lists and readable pages, driven by the keyboard. It knows nothing about
// SSH or timing; the ui package feeds it keys and window sizes.
package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

// menuPaths is the home menu, top to bottom; paths missing from the vfs are
// left out.
var menuPaths = []struct{ label, path string }{
	{"About", "~/about.md"},
	{"Work", "~/work"},
	{"Projects", "~/projects"},
	{"Stack", "~/stack.md"},
	{"Contact", "~/contact.md"},
}

const (
	minW, minH = 20, 8 // smaller windows only get tooSmall
	maxBody    = 80    // widest text column
	tooSmall   = "make the window a bit bigger"

	homeHints   = "↑↓ move · ↵ open · q quit"
	listHints   = "↑↓ move · ↵ open · esc back · q quit"
	detailHints = "↑↓ scroll · esc back · q quit"
)

type kind int

const (
	kindHome kind = iota
	kindList
	kindDetail
)

// page is one screen on the stack.
type page struct {
	kind   kind
	title  string    // list pages: the name shown above the entries
	node   *vfs.Node // list: the directory; detail: the file
	cursor int       // home and list: the selected row
	offset int       // list and detail: the first visible line
	lines  []string  // detail: the page rendered at the current width
}

// Result tells the caller what a key did.
type Result struct {
	Quit   bool
	Opened string // vfs path of the page just opened, if any
}

// Model is one visitor's TUI: a stack of pages with home at the bottom.
type Model struct {
	profile content.Profile
	menu    []menuItem
	stack   []page
	w, h    int
}

// New builds the home page for a w×h window.
func New(fsys *vfs.FS, p content.Profile, w, h int) Model {
	m := Model{profile: p, stack: []page{{kind: kindHome}}}
	for _, it := range menuPaths {
		if n, err := fsys.Resolve(vfs.Home, it.path); err == nil {
			m.menu = append(m.menu, menuItem{label: it.label, node: n})
		}
	}
	m.SetSize(w, h)
	return m
}

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

// bodyWidth is the text column: the window less a margin, at most maxBody.
func (m Model) bodyWidth() int { return min(m.width()-2, maxBody) }

// rows is how many page lines fit between the top bar and the footer.
func (m Model) rows() int { return max(1, m.height()-3) }

// lineCount is how many lines a list or detail page has.
func (m Model) lineCount(p page) int {
	if p.kind == kindList {
		return listLen(p.node)
	}
	return len(p.lines)
}

// SetSize re-renders pages for a new window size and keeps scroll positions
// in range, with each list's selected row still visible.
func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h
	m.stack = slices.Clone(m.stack)
	for i := range m.stack {
		p := &m.stack[i]
		switch p.kind {
		case kindDetail:
			p.lines = detailLines(p.node, m.bodyWidth())
		case kindList:
			p.offset = follow(listHeader+p.cursor, p.offset, m.rows())
		}
		p.offset = clampOffset(p.offset, m.lineCount(*p), m.rows())
	}
}

func isOpen(key string) bool { return key == "enter" || key == "right" || key == "l" }

func isBack(key string) bool {
	return key == "esc" || key == "left" || key == "h" || key == "backspace"
}

// Update handles one key.
func (m Model) Update(k tea.KeyPressMsg) (Model, Result) {
	key := k.String()
	if key == "q" || key == "ctrl+c" {
		return m, Result{Quit: true}
	}
	m.stack = slices.Clone(m.stack)
	if len(m.stack) > 1 && isBack(key) {
		m.stack = m.stack[:len(m.stack)-1]
		return m, Result{}
	}
	p := &m.stack[len(m.stack)-1]
	rows := m.rows()
	switch p.kind {
	case kindHome:
		p.cursor = move(p.cursor, len(m.menu), key, len(m.menu))
		if isOpen(key) && len(m.menu) > 0 {
			it := m.menu[p.cursor]
			return m.open(it.label, it.node)
		}
	case kindList:
		kids := p.node.Children()
		p.cursor = move(p.cursor, len(kids), key, rows)
		if p.cursor == 0 {
			p.offset = 0 // show the title again at the top
		} else {
			p.offset = follow(listHeader+p.cursor, p.offset, rows)
		}
		if isOpen(key) && len(kids) > 0 {
			c := kids[p.cursor]
			return m.open(entryTitle(c), c)
		}
	case kindDetail:
		p.offset = move(p.offset, maxOffset(len(p.lines), rows)+1, key, rows)
	}
	return m, Result{}
}

// open pushes the page for n: a list for a directory, a detail page for a file.
func (m Model) open(title string, n *vfs.Node) (Model, Result) {
	pg := page{kind: kindList, title: title, node: n}
	if !n.Dir {
		pg = page{kind: kindDetail, node: n, lines: detailLines(n, m.bodyWidth())}
	}
	m.stack = append(m.stack, pg)
	return m, Result{Opened: n.Path}
}

// View is the top page, exactly the window's height, every row cut to its width.
func (m Model) View() string {
	w, h := m.width(), m.height()
	if w < minW || h < minH {
		return ansi.Truncate(tooSmall, w, "")
	}
	p := m.stack[len(m.stack)-1]
	if p.kind == kindHome {
		out := fill(homeLines(m.profile, m.menu, p.cursor, w, h), h-1)
		return strings.Join(append(out, center([]string{style.Faint.Render(homeHints)}, w)...), "\n")
	}

	lines, hints := p.lines, detailHints
	if p.kind == kindList {
		lines, hints = listLines(p.title, p.node, p.cursor, m.bodyWidth()), listHints
	}
	rows := m.rows()
	out := []string{
		" " + style.Faint.Render("‹ "+m.backLabel()+"  esc"),
		" " + style.Rule.Render(strings.Repeat("─", w-2)),
	}
	for _, l := range lines[p.offset:min(len(lines), p.offset+rows)] {
		out = append(out, " "+l)
	}
	out = append(fill(out, h-1), footer(hints, marker(p.offset, len(lines), rows), w))
	for i, l := range out {
		out[i] = ansi.Truncate(l, w, "")
	}
	return strings.Join(out, "\n")
}

// backLabel names the page that esc returns to.
func (m Model) backLabel() string {
	if prev := m.stack[len(m.stack)-2]; prev.kind == kindList {
		return prev.title
	}
	return "Home"
}

// fill pads lines with blank rows, or cuts them, to exactly n rows.
func fill(lines []string, n int) []string {
	lines = lines[:min(len(lines), n)]
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

// footer is the key hints on the left and the scroll marker on the right;
// the marker is left out when the window is too narrow for both.
func footer(hints, mark string, w int) string {
	left := " " + style.Faint.Render(hints)
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(mark) - 1
	if mark == "" || gap < 2 {
		return left
	}
	return left + strings.Repeat(" ", gap) + style.Faint.Render(mark)
}
