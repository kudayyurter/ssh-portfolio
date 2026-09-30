package tui

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/kudayyurter/termfolio/content"
	"github.com/kudayyurter/termfolio/internal/pixelfont"
	"github.com/kudayyurter/termfolio/internal/style"
	"github.com/kudayyurter/termfolio/internal/vfs"
)

// menuItem is one home menu entry and the page it opens.
type menuItem struct {
	label string
	node  *vfs.Node
}

// linkIcons are Nerd Font glyphs for profile links, keyed by lowercase label.
// The label is printed too, so fonts without the glyphs still read clearly.
var linkIcons = map[string]string{
	"web":      "", // globe
	"github":   "",
	"linkedin": "",
	"email":    "", // envelope
}

// monogram is "KY" in the site's pixel font at 2×: 22 columns × 7 rows.
var monogram = pixelfont.HalfBlocks(pixelfont.Scale(pixelfont.MustText("KY"), 2))

// wideHome is the narrowest window that shows the monogram beside the name.
const wideHome = 72

// homeLines lays out the home card and menu for a w×h window, leaving the
// last two rows for the footer. When the card does not fit it drops the
// monogram, then the links, then the name and tagline; the menu always stays.
func homeLines(p content.Profile, menu []menuItem, cursor, w, h int) []string {
	room := h - 2
	menuBlock := center(menuRows(menu, cursor), w)
	for _, card := range cards(p, w) {
		block := center(card, w)
		if len(card) > 0 {
			block = append(block, "")
		}
		block = append(block, menuBlock...)
		if len(block) <= room || len(card) == 0 {
			top := max(0, (room-len(block))/2)
			return append(make([]string, top), block...)
		}
	}
	return nil // unreachable: the last card is empty
}

// cards are the home card's layouts, largest first; the last one is empty.
func cards(p content.Profile, w int) [][]string {
	name := []string{style.Heading.Render(p.Name), style.Muted.Render(p.Tagline)}
	var out [][]string
	if w >= wideHome {
		info := append(slices.Clone(name), "")
		out = append(out, beside(monogram, append(info, linkGrid(p.Links, 2)...)))
	}
	full := slices.Clone(name)
	if links := linkGrid(p.Links, 1); len(links) > 0 {
		full = append(append(full, ""), links...)
	}
	return append(out, full, name, nil)
}

// beside puts right to the right of left, top-aligned, four columns apart.
func beside(left, right []string) []string {
	joined := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(left, "\n"), "    ", strings.Join(right, "\n"))
	lines := strings.Split(joined, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

// linkGrid is the profile links, perRow to a row, each an icon and lowercase
// label that is a hyperlink to the link's URL.
func linkGrid(links []content.Link, perRow int) []string {
	cells := make([]string, len(links))
	cellW := 0
	for i, l := range links {
		name := strings.ToLower(l.Label)
		icon := linkIcons[name]
		if icon == "" {
			icon = "•"
		}
		cells[i] = icon + " " + name
		cellW = max(cellW, ansi.StringWidth(cells[i]))
	}
	var rows []string
	for i := 0; i < len(cells); i += perRow {
		row := ""
		for j := i; j < min(i+perRow, len(cells)); j++ {
			if j > i {
				row += strings.Repeat(" ", cellW-ansi.StringWidth(cells[j-1])+4)
			}
			row += style.Link(links[j].URL, style.URL.Render(cells[j]))
		}
		rows = append(rows, row)
	}
	return rows
}

// menuRows is the menu with ▸ and a highlight on the selected entry.
func menuRows(menu []menuItem, cursor int) []string {
	rows := make([]string, len(menu))
	for i, it := range menu {
		if i == cursor {
			rows[i] = style.Path.Render("▸ ") + style.Heading.Render(it.label)
		} else {
			rows[i] = "  " + it.label
		}
	}
	return rows
}

// center shifts a block of lines right so it sits in the middle of w
// columns, keeping its own left alignment, and cuts lines to w.
func center(lines []string, w int) []string {
	bw := 0
	for _, l := range lines {
		bw = max(bw, ansi.StringWidth(l))
	}
	pad := strings.Repeat(" ", max(0, (w-bw)/2))
	out := make([]string, len(lines))
	for i, l := range lines {
		if l != "" {
			l = pad + l
		}
		out[i] = ansi.Truncate(l, w, "")
	}
	return out
}
