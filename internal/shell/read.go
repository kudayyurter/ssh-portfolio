package shell

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kudayyurter/termfolio/internal/style"
	"github.com/kudayyurter/termfolio/internal/vfs"
)

var errIsDir = errors.New("Is a directory")

// maxCatFiles bounds how much one command can print.
const maxCatFiles = 10

func init() {
	register("cat", command{run: catAs("cat"), group: "Reading", usage: "cat <file>", about: "read a file"})
	register("less", command{run: catAs("less")})
	register("more", command{run: catAs("more")})
	register("open", command{run: runOpen, group: "Reading", usage: "open <file>", about: "show a file's link"})
}

// catAs builds cat under a given name so less and more report errors as themselves.
func catAs(name string) func(*Session, []string) Result {
	return func(s *Session, args []string) Result {
		_, ops, err := parseFlags(name, args, "")
		if err != nil {
			return fail(err.Error())
		}
		if len(ops) == 0 {
			return fail(name + ": missing file operand")
		}
		if len(ops) > maxCatFiles {
			return fail(fmt.Sprintf("%s: too many files (at most %d)", name, maxCatFiles))
		}
		var parts []string
		code := 0
		for _, p := range ops {
			n, err := s.fs.Resolve(s.cwd, p)
			if err == nil && n.Dir {
				err = errIsDir
			}
			if err != nil {
				parts = append(parts, style.Err.Render(fmt.Sprintf("%s: %s: %s", name, p, err)))
				code = 1
				continue
			}
			parts = append(parts, s.renderFile(n))
		}
		return Result{Output: strings.Join(parts, "\n\n"), Code: code}
	}
}

// renderFile is a compact header (title; date · stack; link) above the body.
func (s *Session) renderFile(n *vfs.Node) string {
	var b strings.Builder
	b.WriteString(style.Heading.Render(n.Meta.Title))
	var meta []string
	for _, v := range []string{n.Meta.Date, n.Meta.Stack} {
		if v != "" {
			meta = append(meta, v)
		}
	}
	if len(meta) > 0 {
		b.WriteString("\n" + style.Muted.Render(strings.Join(meta, " · ")))
	}
	if n.Meta.Link != "" {
		b.WriteString("\n" + style.Link(n.Meta.Link, style.URL.Render(style.LinkLabel(n.Meta.Link))))
	}
	body, err := style.Markdown(n.Body, s.width)
	if err != nil {
		body = n.Body
	}
	if body != "" {
		b.WriteString("\n\n" + body)
	}
	return b.String()
}

func runOpen(s *Session, args []string) Result {
	if len(args) != 1 {
		return fail("usage: open <file>")
	}
	n, err := s.fs.Resolve(s.cwd, args[0])
	if err != nil {
		return pathErr("open", args[0], err)
	}
	if n.Dir || n.Meta.Link == "" {
		return fail(fmt.Sprintf("open: %s: no link", args[0]))
	}
	return ok(style.Link(n.Meta.Link, n.Meta.Link))
}
