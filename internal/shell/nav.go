package shell

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/kudayyurter/termfolio/internal/style"
	"github.com/kudayyurter/termfolio/internal/vfs"
)

func init() {
	register("ls", command{run: runLs, group: "Moving around", usage: "ls [-la] [path]", about: "list files (-l for details)"})
	register("cd", command{run: runCd, group: "Moving around", usage: "cd [path]", about: "change directory (~, .., -)"})
	register("pwd", command{run: runPwd, group: "Moving around", usage: "pwd", about: "print the current directory"})
	register("tree", command{run: runTree, group: "Moving around", usage: "tree [path]", about: "show everything at once"})
}

func pathErr(cmd, p string, err error) Result {
	return fail(fmt.Sprintf("%s: %s: %s", cmd, p, err))
}

// entry is one line of ls output; "." and ".." are entries without a node.
type entry struct {
	name string
	node *vfs.Node
}

func displayName(e entry) string {
	if e.node == nil || e.node.Dir {
		return style.Dir.Render(e.name + "/")
	}
	return style.File.Render(e.name)
}

func runLs(s *Session, args []string) Result {
	fl, ops, err := parseFlags("ls", args, "la")
	if err != nil {
		return fail(err.Error())
	}
	if len(ops) == 0 {
		ops = []string{"."}
	}
	var blocks []string
	code := 0
	for _, p := range ops {
		n, err := s.fs.Resolve(s.cwd, p)
		if err != nil {
			blocks = append(blocks, style.Err.Render(fmt.Sprintf("ls: %s: %s", p, err)))
			code = 1
			continue
		}
		var entries []entry
		if n.Dir {
			if fl['a'] {
				entries = append(entries, entry{name: "."}, entry{name: ".."})
			}
			for _, c := range n.Children() {
				entries = append(entries, entry{name: c.Name, node: c})
			}
		} else {
			entries = []entry{{name: p, node: n}}
		}
		var body string
		if fl['l'] {
			body = longListing(entries)
		} else {
			body = columns(entries, s.width)
		}
		if len(ops) > 1 && n.Dir {
			body = p + ":\n" + body
		}
		blocks = append(blocks, body)
	}
	return Result{Output: strings.Join(blocks, "\n\n"), Code: code}
}

// columns fills lines left to right, two spaces apart, wrapping at width.
func columns(entries []entry, width int) string {
	if width <= 0 {
		width = 80
	}
	var lines []string
	line, lineW := "", 0
	for _, e := range entries {
		name := displayName(e)
		w := lipgloss.Width(name)
		if lineW > 0 && lineW+2+w > width {
			lines = append(lines, line)
			line, lineW = "", 0
		}
		if lineW > 0 {
			line += "  "
			lineW += 2
		}
		line += name
		lineW += w
	}
	return strings.Join(append(lines, line), "\n")
}

func longListing(entries []entry) string {
	nameW := 0
	for _, e := range entries {
		nameW = max(nameW, lipgloss.Width(displayName(e)))
	}
	var lines []string
	for _, e := range entries {
		perm, info := "drwxr-xr-x", ""
		switch {
		case e.node == nil:
		case e.node.Dir:
			info = style.Muted.Render(countItems(len(e.node.Children())))
		default:
			perm = "-rw-r--r--"
			info = style.Muted.Render(e.node.Meta.Summary)
			if e.node.Meta.Date != "" {
				info += style.Faint.Render(" · " + e.node.Meta.Date)
			}
		}
		name := displayName(e)
		pad := strings.Repeat(" ", nameW-lipgloss.Width(name))
		lines = append(lines, strings.TrimRight(perm+"  "+name+pad+"  "+info, " "))
	}
	return strings.Join(lines, "\n")
}

func countItems(n int) string {
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}

func runCd(s *Session, args []string) Result {
	if len(args) > 1 {
		return fail("cd: too many arguments")
	}
	target, announce := "~", false
	if len(args) == 1 {
		target = args[0]
	}
	if target == "-" {
		if s.prev == "" {
			return fail("cd: OLDPWD not set")
		}
		target, announce = s.prev, true
	}
	n, err := s.fs.Resolve(s.cwd, target)
	if err != nil {
		return pathErr("cd", target, err)
	}
	if !n.Dir {
		return pathErr("cd", target, vfs.ErrNotDir)
	}
	s.prev, s.cwd = s.cwd, n.Path
	if announce {
		return ok(vfs.Display(s.cwd))
	}
	return Result{}
}

func runPwd(s *Session, _ []string) Result { return ok(s.cwd) }

func runTree(s *Session, args []string) Result {
	_, ops, err := parseFlags("tree", args, "")
	if err != nil {
		return fail(err.Error())
	}
	if len(ops) > 1 {
		return fail("tree: too many arguments")
	}
	label := "."
	if len(ops) == 1 {
		label = ops[0]
	}
	n, err := s.fs.Resolve(s.cwd, label)
	if err != nil {
		return pathErr("tree", label, err)
	}
	if !n.Dir {
		return ok(label)
	}
	var b strings.Builder
	b.WriteString(style.Dir.Render(label))
	dirs, files := 0, 0
	var walk func(n *vfs.Node, indent string)
	walk = func(n *vfs.Node, indent string) {
		kids := n.Children()
		for i, c := range kids {
			branch, next := "├── ", "│   "
			if i == len(kids)-1 {
				branch, next = "└── ", "    "
			}
			b.WriteString("\n" + style.Rule.Render(indent+branch) + displayName(entry{name: c.Name, node: c}))
			if c.Dir {
				dirs++
				walk(c, indent+next)
			} else {
				files++
			}
		}
	}
	walk(n, "")
	fmt.Fprintf(&b, "\n\n%s", style.Muted.Render(fmt.Sprintf("%d directories, %d files", dirs, files)))
	return ok(b.String())
}
