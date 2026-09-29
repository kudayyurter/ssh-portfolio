package tui

import (
	"strings"

	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

// detailLines is a file's page: the title, "date · stack" and the link when
// the file has them, then the rendered markdown body wrapped at width.
func detailLines(n *vfs.Node, width int) []string {
	lines := []string{style.Heading.Render(n.Meta.Title)}
	var meta []string
	for _, v := range []string{n.Meta.Date, n.Meta.Stack} {
		if v != "" {
			meta = append(meta, v)
		}
	}
	if len(meta) > 0 {
		lines = append(lines, style.Muted.Render(strings.Join(meta, " · ")))
	}
	if n.Meta.Link != "" {
		lines = append(lines, style.Link(n.Meta.Link, style.URL.Render(style.LinkLabel(n.Meta.Link))))
	}
	body, err := style.Markdown(n.Body, width)
	if err != nil {
		body = n.Body
	}
	if body != "" {
		lines = append(lines, "")
		lines = append(lines, strings.Split(body, "\n")...)
	}
	return lines
}
