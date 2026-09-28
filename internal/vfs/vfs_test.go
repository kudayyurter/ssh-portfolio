package vfs

import (
	"errors"
	"testing"
	"testing/fstest"
)

func testFS(t *testing.T) *FS {
	t.Helper()
	fsys, err := New(fstest.MapFS{
		"about.md":            {Data: []byte("---\ntitle: About\nsummary: Who I am\norder: 1\n---\nHello.\n")},
		"contact.md":          {Data: []byte("---\ntitle: Contact\nsummary: Reach me\norder: 2\n---\nEmail.\n")},
		"work/acme.md":        {Data: []byte("---\ntitle: Acme\nsummary: Engineer\ndate: 2025 — Present\norder: 2\n---\nBuilt.\n")},
		"work/zeta.md":        {Data: []byte("---\ntitle: Zeta\nsummary: Intern\norder: 1\n---\nLearned.\n")},
		"projects/kessler.md": {Data: []byte("---\ntitle: Kessler\nsummary: Orbit globe\nlink: https://k.example\n---\nGlobe.\n")},
		"projects/alpha.md":   {Data: []byte("---\ntitle: Alpha\nsummary: First\n---\nA.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func TestResolve(t *testing.T) {
	fsys := testFS(t)
	cases := []struct {
		cwd, path, want string
		err             error
	}{
		{Home, "", Home, nil},
		{Home, ".", Home, nil},
		{Home, "~", Home, nil},
		{"/", "~/work", Home + "/work", nil},
		{Home, "work/acme.md", Home + "/work/acme.md", nil},
		{Home + "/work", "..", Home, nil},
		{Home + "/work", "../about.md", Home + "/about.md", nil},
		{Home, "./work/", Home + "/work", nil},
		{Home, "/", "/", nil},
		{Home, "/home", "/home", nil},
		{Home, "../../../../..", "/", nil},
		{Home, "../../../../etc/passwd", "", ErrNotExist},
		{Home, "nope", "", ErrNotExist},
		{Home, "about.md/x", "", ErrNotDir},
		{Home, "about.md/", "", ErrNotDir},
	}
	for _, c := range cases {
		n, err := fsys.Resolve(c.cwd, c.path)
		if c.err != nil {
			if !errors.Is(err, c.err) {
				t.Errorf("Resolve(%q, %q) err = %v, want %v", c.cwd, c.path, err, c.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q, %q) unexpected err %v", c.cwd, c.path, err)
			continue
		}
		if n.Path != c.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", c.cwd, c.path, n.Path, c.want)
		}
	}
}

func TestChildrenOrder(t *testing.T) {
	fsys := testFS(t)
	home, _ := fsys.Resolve(Home, "")
	var got []string
	for _, c := range home.Children() {
		got = append(got, c.Name)
	}
	want := []string{"projects", "work", "about.md", "contact.md"}
	if len(got) != len(want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("children = %v, want %v", got, want)
		}
	}

	work, _ := fsys.Resolve(Home, "work")
	if w := work.Children(); w[0].Name != "zeta.md" || w[1].Name != "acme.md" {
		t.Fatalf("work not sorted by order: %s, %s", w[0].Name, w[1].Name)
	}
	projects, _ := fsys.Resolve(Home, "projects")
	if p := projects.Children(); p[0].Name != "alpha.md" || p[1].Name != "kessler.md" {
		t.Fatalf("projects without order not alphabetical: %s, %s", p[0].Name, p[1].Name)
	}
}

func TestFrontMatter(t *testing.T) {
	fsys := testFS(t)
	n, _ := fsys.Resolve(Home, "work/acme.md")
	if n.Meta.Title != "Acme" || n.Meta.Date != "2025 — Present" || n.Meta.Order != 2 {
		t.Fatalf("meta = %+v", n.Meta)
	}
	if n.Body != "Built.\n" {
		t.Fatalf("body = %q", n.Body)
	}
}

func TestNewRejectsBadContent(t *testing.T) {
	cases := map[string]string{
		"no front matter":  "hello\n",
		"unterminated":     "---\ntitle: X\nsummary: Y\n",
		"missing summary":  "---\ntitle: X\n---\nbody\n",
		"unknown field":    "---\ntitle: X\nsummary: Y\ncolor: red\n---\nbody\n",
		"not yaml mapping": "---\n- a\n---\nbody\n",
	}
	for name, data := range cases {
		if _, err := New(fstest.MapFS{"x.md": {Data: []byte(data)}}); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDisplay(t *testing.T) {
	cases := map[string]string{
		Home:              "~",
		Home + "/work":    "~/work",
		"/home":           "/home",
		"/":               "/",
		"/home/guestbook": "/home/guestbook",
	}
	for in, want := range cases {
		if got := Display(in); got != want {
			t.Errorf("Display(%q) = %q, want %q", in, got, want)
		}
	}
}
