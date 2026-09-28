// Package style is the portfolio's look: a monochrome palette matching
// kudayyurter.dev, hyperlinks, and markdown rendering. Plain text always uses
// the terminal's default foreground so light and dark terminals both work.
package style

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Palette (from the site's CSS tokens).
var (
	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#a3a3a3"))
	Faint = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	Rule  = lipgloss.NewStyle().Foreground(lipgloss.Color("#262626"))
	Bold  = lipgloss.NewStyle().Bold(true)
)

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
