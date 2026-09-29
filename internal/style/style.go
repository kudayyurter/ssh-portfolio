// Package style is the portfolio's look: the tokyodark palette, hyperlinks,
// and markdown rendering. Plain text always uses the terminal's default
// foreground; color marks structure (prompt, names, headings, links, errors).
package style

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Palette: tokyodark (github.com/tiagovla/tokyodark.nvim), assigned the way
// oddbit.ai colors its terminal page.
var (
	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A8CD"))
	Faint = lipgloss.NewStyle().Foreground(lipgloss.Color("#737AA2"))
	Rule  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5057"))
	Bold  = lipgloss.NewStyle().Bold(true)

	User    = lipgloss.NewStyle().Foreground(lipgloss.Color("#95C561"))
	Host    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7199EE"))
	Path    = lipgloss.NewStyle().Foreground(lipgloss.Color("#D7A65F"))
	Heading = lipgloss.NewStyle().Foreground(lipgloss.Color("#A485DD")).Bold(true)
	Label   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7199EE")).Bold(true)
	Dir     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7199EE")).Bold(true)
	File    = lipgloss.NewStyle().Foreground(lipgloss.Color("#D7A65F"))
	Command = lipgloss.NewStyle().Foreground(lipgloss.Color("#95C561"))
	URL     = lipgloss.NewStyle().Foreground(lipgloss.Color("#38A89D"))
	Err     = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6D85"))
)

// UserHost is "guest@kuday" in the prompt's colors.
func UserHost() string {
	return User.Render("guest") + Muted.Render("@") + Host.Render("kuday")
}

// Link makes text an OSC 8 hyperlink to url; terminals without support show
// the text unchanged.
func Link(url, text string) string {
	return ansi.SetHyperlink(url) + text + ansi.ResetHyperlink()
}

// LinkLabel is how a URL is shown to people: no scheme, no trailing slash.
func LinkLabel(url string) string {
	for _, p := range []string{"https://", "http://", "mailto:"} {
		url = strings.TrimPrefix(url, p)
	}
	url = strings.TrimPrefix(url, "www.")
	return strings.TrimSuffix(url, "/")
}
