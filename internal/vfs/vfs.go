// Package vfs is the read-only virtual filesystem visitors explore. It is
// built once from markdown files with YAML front matter and mounted at
// /home/guest; nothing outside it exists.
package vfs

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Home is the visitor's home directory.
const Home = "/home/guest"

// Errors carry the exact wording shells print after "<cmd>: <path>: ".
var (
	ErrNotExist = errors.New("No such file or directory")
	ErrNotDir   = errors.New("Not a directory")
)

// Meta is a file's front matter.
type Meta struct {
	Title   string `yaml:"title"`
	Summary string `yaml:"summary"`
	Date    string `yaml:"date"`
	Stack   string `yaml:"stack"`
	Link    string `yaml:"link"`
	Order   int    `yaml:"order"`
}

// Node is a file or directory.
type Node struct {
	Name     string // base name, "" for the root
	Path     string // absolute, e.g. /home/guest/work/cummins.md
	Dir      bool
	Meta     Meta   // files only
	Body     string // markdown after the front matter; files only
	children []*Node
}

// Children returns a directory's entries: directories first (alphabetical),
// then files by Meta.Order (files without an order last, alphabetical).
func (n *Node) Children() []*Node { return n.children }

func (n *Node) child(name string) *Node {
	for _, c := range n.children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// FS is the whole tree.
type FS struct{ root *Node }

// New builds the tree from src, whose root becomes /home/guest. Only .md files
// are loaded; every one must have valid front matter with a title and summary.
func New(src fs.FS) (*FS, error) {
	root := &Node{Path: "/", Dir: true}
	home := &Node{Name: "home", Path: "/home", Dir: true}
	guest := &Node{Name: "guest", Path: Home, Dir: true}
	root.children = []*Node{home}
	home.children = []*Node{guest}

	dirs := map[string]*Node{".": guest}
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		parent := dirs[path.Dir(p)]
		if d.IsDir() {
			n := &Node{Name: d.Name(), Path: Home + "/" + p, Dir: true}
			parent.children = append(parent.children, n)
			dirs[p] = n
			return nil
		}
		if path.Ext(p) != ".md" {
			return nil
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		meta, body, err := parse(p, data)
		if err != nil {
			return err
		}
		parent.children = append(parent.children, &Node{Name: d.Name(), Path: Home + "/" + p, Meta: meta, Body: body})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortTree(root)
	return &FS{root: root}, nil
}

func sortTree(n *Node) {
	slices.SortStableFunc(n.children, func(a, b *Node) int {
		switch {
		case a.Dir != b.Dir:
			if a.Dir {
				return -1
			}
			return 1
		case a.Dir:
			return strings.Compare(a.Name, b.Name)
		}
		ao, bo := a.Meta.Order, b.Meta.Order
		if (ao == 0) != (bo == 0) { // ordered files come before unordered ones
			if ao == 0 {
				return 1
			}
			return -1
		}
		return cmp.Or(cmp.Compare(ao, bo), strings.Compare(a.Name, b.Name))
	})
	for _, c := range n.children {
		if c.Dir {
			sortTree(c)
		}
	}
}

func parse(name string, data []byte) (Meta, string, error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return Meta{}, "", fmt.Errorf("%s: missing front matter", name)
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return Meta{}, "", fmt.Errorf("%s: unterminated front matter", name)
	}
	head, body := s[4:4+end+1], s[4+end+5:]

	var m Meta
	dec := yaml.NewDecoder(strings.NewReader(head))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return Meta{}, "", fmt.Errorf("%s: %w", name, err)
	}
	if m.Title == "" || m.Summary == "" {
		return Meta{}, "", fmt.Errorf("%s: title and summary are required", name)
	}
	return m, strings.TrimLeft(body, "\n"), nil
}

// Resolve finds p relative to cwd. It understands ~, ~/x, absolute paths,
// . and ..; going above / stays at /. A trailing slash on a file is ErrNotDir.
func (f *FS) Resolve(cwd, p string) (*Node, error) {
	var abs string
	switch {
	case p == "":
		abs = cwd
	case p == "~":
		abs = Home
	case strings.HasPrefix(p, "~/"):
		abs = Home + p[1:]
	case strings.HasPrefix(p, "/"):
		abs = p
	default:
		abs = cwd + "/" + p
	}
	abs = path.Clean(abs)

	n := f.root
	if abs != "/" {
		for _, part := range strings.Split(abs[1:], "/") {
			if !n.Dir {
				return nil, ErrNotDir
			}
			if n = n.child(part); n == nil {
				return nil, ErrNotExist
			}
		}
	}
	if !n.Dir && strings.HasSuffix(p, "/") {
		return nil, ErrNotDir
	}
	return n, nil
}

// Display shortens an absolute path for prompts: /home/guest/work → ~/work.
func Display(abs string) string {
	switch {
	case abs == Home:
		return "~"
	case strings.HasPrefix(abs, Home+"/"):
		return "~" + abs[len(Home):]
	}
	return abs
}
