package style

import (
	_ "embed"
	"strings"

	"charm.land/glamour/v2"
	"github.com/charmbracelet/x/ansi"
)

//go:embed theme.json
var theme []byte

// renderer is built once; glamour renderers are safe to reuse.
var renderer = func() *glamour.TermRenderer {
	// Glamour's own wrapping breaks inside words like "4.9/5", so it is
	// turned off (0) and lines are wrapped below at spaces only.
	r, err := glamour.NewTermRenderer(glamour.WithStylesFromJSONBytes(theme), glamour.WithWordWrap(0))
	if err != nil {
		panic(err) // theme.json is embedded; a bad theme is a build-time bug
	}
	return r
}()

// Markdown renders md to wrap at width columns. Unknown or tiny widths fall
// back to 80; very wide terminals are capped at 100 so lines stay readable.
func Markdown(md string, width int) (string, error) {
	switch {
	case width < 20:
		width = 80
	case width > 100:
		width = 100
	}
	out, err := renderer.Render(md)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		lines = append(lines, wrap(strings.TrimRight(l, " "), width)...)
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n"), nil
}

// wrap breaks one rendered line at spaces (hard-breaking words longer than
// the width). Continuation lines of a bullet are indented under its text.
func wrap(line string, width int) []string {
	if ansi.StringWidth(line) <= width {
		return []string{line}
	}
	plain := ansi.Strip(line)
	indent := len(plain) - len(strings.TrimLeft(plain, " "))
	if strings.HasPrefix(plain[indent:], "• ") {
		indent += 2
	}
	parts := strings.Split(ansi.Wrap(line, width-indent, ""), "\n")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.Repeat(" ", indent) + strings.TrimLeft(parts[i], " ")
	}
	return parts
}
