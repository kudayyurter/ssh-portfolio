package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/kudayyurter/ssh-portfolio/internal/style"
	"github.com/kudayyurter/ssh-portfolio/internal/vfs"
)

// listHeader is how many rows sit above a list's first entry: title, blank.
const listHeader = 2

// maxTitle caps the title column so summaries keep room on narrow windows.
const maxTitle = 24

// entryTitle is how an entry is named in a list.
func entryTitle(n *vfs.Node) string {
	if n.Dir {
		return n.Name + "/"
	}
	return n.Meta.Title
}

// entryInfo is the muted text after an entry's title.
func entryInfo(n *vfs.Node) string {
	if n.Dir {
		if k := len(n.Children()); k != 1 {
			return fmt.Sprintf("%d items", k)
		}
		return "1 item"
	}
	if n.Meta.Date == "" {
		return n.Meta.Summary
	}
	return n.Meta.Summary + " · " + n.Meta.Date
}

// listLen is how many lines listLines returns for dir.
func listLen(dir *vfs.Node) int { return listHeader + max(1, len(dir.Children())) }

// listLines is a directory's page: its title, then one row per entry with
// the title in a column and "summary · date" after it. The row at cursor is
// marked with ▸ and its title highlighted. Rows are cut to width.
func listLines(title string, dir *vfs.Node, cursor, width int) []string {
	lines := []string{style.Heading.Render(title), ""}
	kids := dir.Children()
	if len(kids) == 0 {
		return append(lines, style.Muted.Render("nothing here yet"))
	}
	titleW := 0
	for _, c := range kids {
		titleW = max(titleW, ansi.StringWidth(entryTitle(c)))
	}
	titleW = min(titleW, maxTitle)
	for i, c := range kids {
		mark, name := "  ", ansi.Truncate(entryTitle(c), titleW, "…")
		pad := strings.Repeat(" ", titleW-ansi.StringWidth(name))
		if i == cursor {
			mark, name = style.Path.Render("▸ "), style.Heading.Render(name)
		}
		row := mark + name
		if info := entryInfo(c); info != "" {
			row += pad + "  " + style.Muted.Render(info)
		}
		lines = append(lines, ansi.Truncate(row, width, "…"))
	}
	return lines
}
