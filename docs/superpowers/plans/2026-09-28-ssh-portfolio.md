# SSH Portfolio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Go SSH server where `ssh term.kudayyurter.dev` plays a centered pixel-name boot animation, then drops the visitor into a read-only, shell-like portfolio (`ls`, `cd`, `cat`, `tree`, …) backed by embedded markdown.

**Architecture:** One static Go binary. Charm Wish is the SSH server; every PTY session gets its own Bubble Tea program (`internal/ui`) that shows the boot animation (`internal/boot`, a pure frame function) in the alt screen, then runs the shell inline so output lands in the terminal's real scrollback. The shell (`internal/shell`) is terminal-agnostic: it takes a line and returns a styled `Result`, so the same code serves interactive sessions and `ssh host cat about.md`. Content lives in `content/` as markdown with YAML front matter, loaded into a virtual filesystem (`internal/vfs`).

**Tech Stack:** Go 1.27, `charm.land/wish/v2` v2.0.4, `charm.land/bubbletea/v2` v2.0.10, `charm.land/lipgloss/v2` v2.0.6, `charm.land/glamour/v2` v2.0.1, `charm.land/ssh` v0.4.3, `github.com/charmbracelet/colorprofile` v0.4.3, `github.com/charmbracelet/x/ansi`, `gopkg.in/yaml.v3`, `golang.org/x/time/rate`, `golang.org/x/crypto/ssh` (tests). Docker (distroless), AWS Lightsail, GitHub Actions, Vercel DNS.

**Spec:** `docs/superpowers/specs/2026-09-28-ssh-portfolio-design.md`

## Global Constraints

- Module path: `github.com/namelessmonarch0/ssh-portfolio`; `go 1.27` in `go.mod`.
- Visitors authenticate with nothing: no password, no key, nothing stored about keys.
- Read-only: nothing a visitor types changes state beyond their own session (cwd, history).
- Colors: muted `#a3a3a3`, faint `#808080`, lines `#262626`. Body text and headings use the terminal's **default foreground** (never hard-coded `#ffffff`) so light-background terminals stay readable.
- Prompt: `guest@kuday:<path>$ ` where `<path>` is `~`-relative.
- Content copy: plain language, products over jargon, no city/state locations; facts only from `~/DEV/Resume/knowledge_base.md` and the current `~/DEV/Portfolio/src/content/portfolio.ts`. Do not show a graduation date.
- Exit codes: 0 ok, 1 error, 127 command not found.
- Error format: `<cmd>: <path>: No such file or directory` / `Not a directory` / `Is a directory`; unknown flag `<cmd>: invalid option -- 'x'`.
- Boot animation total length `2100ms`; any key skips and is swallowed.
- Limits: 100 sessions global, 5 per IP, rate 1/s burst 10 per IP, idle 10m, max session 30m.
- Env vars: `LISTEN_ADDR` (`:2222`), `HOST_KEY_PATH` (`/data/ssh_host_ed25519`), `MAX_SESSIONS` (`100`), `IDLE_TIMEOUT` (`10m`), `MAX_SESSION` (`30m`), `PUBLIC_HOST` (`term.kudayyurter.dev`).
- Commit messages end with:
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18
  ```
- Outward-facing steps (GitHub repo creation, secrets, AWS resources, DNS) are marked **STOP — ask the user** and must not run without an explicit yes in the current conversation.

## Changes from the spec (decided while planning, approved together with this plan)

1. **Native scrollback instead of in-app scrolling.** The boot animation uses the alt screen; the shell then runs inline and prints each command's output with `tea.Println`, so output lives in the terminal's own scrollback. Mouse wheel, Shift+PgUp and text selection/copy work natively (copying the email address matters). The spec's "mouse wheel and PgUp/PgDn scroll" is satisfied by the terminal rather than by the app.
2. **Deploy ships the image over SSH** (`docker save | gzip | ssh deploy@host`) instead of via GHCR: no registry credentials on the server, no package-visibility setup. The deploy key is restricted to a forced command.
3. **Admin port 2200 is open to all IPs, key-only.** GitHub Actions runners have changing IPs, so an IP allowlist would block deploys. Password and keyboard-interactive auth are disabled.
4. **Body text uses the terminal default color** instead of `#ffffff` (light-background terminals).
5. **`content/profile.yaml`** holds identity strings (name, tagline, role, school, degree, top stack, links) used by the boot screen, `whoami`, `neofetch`, and the welcome; `PUBLIC_HOST` env var feeds the `ssh -t` hint.

## Review Focus

1. **Light-background terminals** — a visitor on a white terminal must still read every line: no hard-coded white foreground anywhere (tests in Task 3 and Task 7).
2. **Pasted or typed control characters** — pasting multi-line text or escape sequences into the prompt, or `ssh host $'echo \e[2J'`, must never inject terminal escapes into output; newlines become spaces (tests in Task 4 and Task 8).
3. **Zero or tiny window sizes** — some clients report 0×0 or resize to 10 columns mid-session: nothing panics, markdown falls back to 80 columns, the prompt wraps (tests in Task 3, Task 7, Task 8).
4. **Path escape attempts** — `cat ../../../../etc/passwd`, `cd /home`, `ls /` behave like a tiny real filesystem and never reach the host (tests in Task 2 and Task 4).
5. **Input during boot** — keys or a paste arriving mid-animation skip the animation and are swallowed, never typed into the prompt (test in Task 8).

## File Structure

```
go.mod, go.sum
cmd/server/main.go                 env config, wiring, signals, graceful shutdown
content/embed.go                   //go:embed of markdown + profile.yaml, Profile type
content/profile.yaml               identity strings
content/*.md, work/*.md, projects/*.md   the visitor's home directory
content/content_test.go            content validity + location lint
internal/pixelfont/pixelfont.go    5×7 glyphs (ported from the site), bitmap ops, half-block render
internal/vfs/vfs.go                FS, Node, Meta, Resolve, Children, Display
internal/style/style.go            palette, Link, LinkLabel
internal/style/markdown.go         glamour renderer with embedded theme
internal/style/theme.json          monochrome glamour theme
internal/shell/session.go          Session, Result, Action, Run, Prompt
internal/shell/parse.go            split, sanitize, parseFlags
internal/shell/suggest.go          notFound, closest, distance
internal/shell/nav.go              ls, cd, pwd, tree
internal/shell/read.go             cat/less/more, open
internal/shell/about.go            whoami, neofetch, help
internal/shell/misc.go             clear, history, echo, date, exit/logout, boot, easter eggs
internal/shell/complete.go         Tab completion
internal/boot/boot.go              Frame, Fits, Duration, Options
internal/boot/canvas.go            styled cell grid → string
internal/ui/ui.go                  Bubble Tea model: boot ↔ shell, keys, view
internal/server/config.go          Config, ConfigFromEnv
internal/server/server.go          Server, New, Shutdown, program handler, exec mode, logging
internal/server/limits.go          global + per-IP session limiter
internal/server/registry.go        live programs for shutdown broadcast
Dockerfile, .dockerignore
.github/workflows/ci.yml           test (+ deploy job added in Task 11)
deploy/cloud-init.sh               first-boot setup of the Lightsail box
deploy/lightsail.sh                provisioning script (AWS CLI)
README.md                          local dev + ops runbook
```

---

### Task 1: Module scaffold and pixel font

**Files:**
- Create: `go.mod`, `.gitignore`, `internal/pixelfont/pixelfont.go`
- Test: `internal/pixelfont/pixelfont_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type pixelfont.Bitmap []string` with `func (b Bitmap) Width() int`; `const pixelfont.Height = 7`
  - `func pixelfont.Text(s string) (Bitmap, error)`, `func pixelfont.MustText(s string) Bitmap`
  - `func pixelfont.Join(gap int, parts ...Bitmap) Bitmap` — side by side
  - `func pixelfont.Stack(gap int, parts ...Bitmap) Bitmap` — top to bottom, narrower parts centered
  - `func pixelfont.Scale(b Bitmap, k int) Bitmap`
  - `func pixelfont.HalfBlocks(b Bitmap) []string` — two pixel rows per text row with `▀ ▄ █`

- [ ] **Step 1: Initialize the module** (from `~/DEV/ssh-portfolio`, which already holds the spec commit)

```bash
go mod init github.com/namelessmonarch0/ssh-portfolio
go mod edit -go=1.27
```

Create `.gitignore`:

```gitignore
/.data/
/deploy/keys/
/ssh-portfolio
*.test
```

- [ ] **Step 2: Write the failing test**

Create `internal/pixelfont/pixelfont_test.go`:

```go
package pixelfont

import (
	"slices"
	"testing"
)

func TestTextLaysOutGlyphsWithOneColumnGap(t *testing.T) {
	got, err := Text("KY")
	if err != nil {
		t.Fatal(err)
	}
	want := Bitmap{
		"10001010001",
		"10010010001",
		"10100001010",
		"11000000100",
		"10100000100",
		"10010000100",
		"10001000100",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Text(KY) =\n%v\nwant\n%v", got, want)
	}
}

func TestTextRejectsUnknownGlyph(t *testing.T) {
	if _, err := Text("K?"); err == nil {
		t.Fatal("expected an error for '?'")
	}
}

func TestFullNameHalfBlocks(t *testing.T) {
	name := Join(4, MustText("KUDAY"), MustText("YURTER"))
	if name.Width() != 68 {
		t.Fatalf("width = %d, want 68", name.Width())
	}
	want := []string{
		"█  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █    █   █ █   █ █▀▀▀▄ ▀▀█▀▀ █▀▀▀▀ █▀▀▀▄",
		"█▄▀   █   █ █   █ █▄▄▄█  ▀▄▀      ▀▄▀  █   █ █▄▄▄▀   █   █▄▄▄  █▄▄▄▀",
		"█ ▀▄  █   █ █   █ █   █   █        █   █   █ █ ▀▄    █   █     █ ▀▄ ",
		"▀   ▀  ▀▀▀  ▀▀▀▀  ▀   ▀   ▀        ▀    ▀▀▀  ▀   ▀   ▀   ▀▀▀▀▀ ▀   ▀",
	}
	if got := HalfBlocks(name); !slices.Equal(got, want) {
		t.Fatalf("HalfBlocks =\n%q\nwant\n%q", got, want)
	}
}

func TestStackCentersNarrowerRows(t *testing.T) {
	got := HalfBlocks(Stack(1, MustText("KUDAY"), MustText("YURTER")))
	want := []string{
		"   █  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █   ",
		"   █▄▀   █   █ █   █ █▄▄▄█  ▀▄▀    ",
		"   █ ▀▄  █   █ █   █ █   █   █     ",
		"   ▀   ▀  ▀▀▀  ▀▀▀▀  ▀   ▀   ▀     ",
		"█   █ █   █ █▀▀▀▄ ▀▀█▀▀ █▀▀▀▀ █▀▀▀▄",
		" ▀▄▀  █   █ █▄▄▄▀   █   █▄▄▄  █▄▄▄▀",
		"  █   █   █ █ ▀▄    █   █     █ ▀▄ ",
		"  ▀    ▀▀▀  ▀   ▀   ▀   ▀▀▀▀▀ ▀   ▀",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("stacked =\n%q\nwant\n%q", got, want)
	}
}

func TestScaleDoublesEveryPixel(t *testing.T) {
	got := HalfBlocks(Scale(MustText("KY"), 2))
	want := []string{
		"██      ██  ██      ██",
		"██    ██    ██      ██",
		"██  ██        ██  ██  ",
		"████            ██    ",
		"██  ██          ██    ",
		"██    ██        ██    ",
		"██      ██      ██    ",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("scaled =\n%q\nwant\n%q", got, want)
	}
}
```

The expected strings were computed independently from the glyph table (the same rows the website draws); do not regenerate them from the implementation.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/pixelfont/`
Expected: FAIL — `undefined: Text` (and the other functions).

- [ ] **Step 4: Write the implementation**

Create `internal/pixelfont/pixelfont.go`:

```go
// Package pixelfont holds the 5×7 pixel glyphs shared with kudayyurter.dev
// (Portfolio/src/lib/pixel-font.ts) and turns text into 1-bit bitmaps.
package pixelfont

import (
	"fmt"
	"strings"
)

// Bitmap is a 1-bit image: one string per row, '1' = filled pixel.
type Bitmap []string

// Width is the number of pixel columns.
func (b Bitmap) Width() int {
	if len(b) == 0 {
		return 0
	}
	return len(b[0])
}

// Height of every glyph.
const Height = 7

// Glyphs are copied verbatim from the website so both render the same name.
var Glyphs = map[rune]Bitmap{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "01010", "01010", "00100"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'-': {"00000", "00000", "00000", "01110", "00000", "00000", "00000"},
}

// Text lays glyphs out left to right with a one-column gap.
func Text(s string) (Bitmap, error) {
	rows := make(Bitmap, Height)
	for i, r := range []rune(s) {
		g, ok := Glyphs[r]
		if !ok {
			return nil, fmt.Errorf("pixelfont: no glyph for %q", r)
		}
		for y := range Height {
			if i > 0 {
				rows[y] += "0"
			}
			rows[y] += g[y]
		}
	}
	return rows, nil
}

// MustText is Text for fixed strings known to have glyphs.
func MustText(s string) Bitmap {
	b, err := Text(s)
	if err != nil {
		panic(err)
	}
	return b
}

// Join places bitmaps side by side with gap empty columns between them.
// All parts must have the same height.
func Join(gap int, parts ...Bitmap) Bitmap {
	if len(parts) == 0 {
		return nil
	}
	rows := make(Bitmap, len(parts[0]))
	for i, p := range parts {
		for y := range rows {
			if i > 0 {
				rows[y] += strings.Repeat("0", gap)
			}
			rows[y] += p[y]
		}
	}
	return rows
}

// Stack places bitmaps top to bottom with gap empty rows between them,
// centering narrower parts (extra column goes on the right).
func Stack(gap int, parts ...Bitmap) Bitmap {
	w := 0
	for _, p := range parts {
		w = max(w, p.Width())
	}
	var rows Bitmap
	for i, p := range parts {
		if i > 0 {
			for range gap {
				rows = append(rows, strings.Repeat("0", w))
			}
		}
		left := (w - p.Width()) / 2
		right := w - p.Width() - left
		for _, r := range p {
			rows = append(rows, strings.Repeat("0", left)+r+strings.Repeat("0", right))
		}
	}
	return rows
}

// Scale enlarges every pixel to k×k.
func Scale(b Bitmap, k int) Bitmap {
	var rows Bitmap
	for _, r := range b {
		var sb strings.Builder
		for _, c := range r {
			sb.WriteString(strings.Repeat(string(c), k))
		}
		for range k {
			rows = append(rows, sb.String())
		}
	}
	return rows
}

// HalfBlocks renders the bitmap two pixel rows per text row using ▀ ▄ █,
// which keeps pixels square in a terminal. An odd last row is padded.
func HalfBlocks(b Bitmap) []string {
	rows := b
	if len(rows)%2 == 1 {
		rows = append(slicesClone(rows), strings.Repeat("0", b.Width()))
	}
	out := make([]string, 0, len(rows)/2)
	for y := 0; y < len(rows); y += 2 {
		var sb strings.Builder
		for x := range len(rows[y]) {
			top, bottom := rows[y][x] == '1', rows[y+1][x] == '1'
			switch {
			case top && bottom:
				sb.WriteRune('█')
			case top:
				sb.WriteRune('▀')
			case bottom:
				sb.WriteRune('▄')
			default:
				sb.WriteRune(' ')
			}
		}
		out = append(out, sb.String())
	}
	return out
}

func slicesClone(b Bitmap) Bitmap { return append(Bitmap(nil), b...) }
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/pixelfont/ -v`
Expected: PASS for all five tests.

- [ ] **Step 6: Commit**

```bash
git add go.mod .gitignore internal/pixelfont
git commit -m "feat: add pixel font ported from the website

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 2: Virtual filesystem and portfolio content

**Files:**
- Create: `internal/vfs/vfs.go`, `content/embed.go`, `content/profile.yaml`, and the 12 markdown files under `content/`
- Test: `internal/vfs/vfs_test.go`, `content/content_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `const vfs.Home = "/home/guest"`; `var vfs.ErrNotExist, vfs.ErrNotDir error` (messages `No such file or directory`, `Not a directory`)
  - `type vfs.Meta struct { Title, Summary, Date, Stack, Link string; Order int }`
  - `type vfs.Node struct { Name, Path string; Dir bool; Meta Meta; Body string }` with `func (n *Node) Children() []*Node` (dirs first alphabetical, then files by `Order`, unordered files last alphabetical)
  - `func vfs.New(src fs.FS) (*FS, error)`; `func (f *FS) Resolve(cwd, p string) (*Node, error)`; `func vfs.Display(abs string) string`
  - `var content.Files embed.FS`; `type content.Link struct { Label, URL string }`; `type content.Profile struct { Name, Tagline, Role, School, Degree string; Stack []string; Links []Link }`; `func content.LoadProfile() (Profile, error)`

- [ ] **Step 1: Add the YAML dependency**

```bash
go get gopkg.in/yaml.v3@v3.0.1
```

- [ ] **Step 2: Write the failing vfs test**

Create `internal/vfs/vfs_test.go`:

```go
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
```

The `Resolve` table pins Review Focus 4 (`../../../../etc/passwd` is `ErrNotExist`; going above `/` stays at `/`).

- [ ] **Step 3: Run it to verify it fails**

Run: `go test ./internal/vfs/`
Expected: FAIL — `undefined: New`, `undefined: Home`, etc.

- [ ] **Step 4: Implement the filesystem**

Create `internal/vfs/vfs.go`:

```go
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
```

- [ ] **Step 5: Run the vfs tests**

Run: `go test ./internal/vfs/ -v`
Expected: PASS (TestResolve, TestChildrenOrder, TestFrontMatter, TestNewRejectsBadContent, TestDisplay).

- [ ] **Step 6: Write the content test (fails: no content package yet)**

Create `content/content_test.go`:

```go
package content

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func TestFilesBuildAValidTree(t *testing.T) {
	fsys, err := vfs.New(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"about.md", "contact.md", "stack.md", "work/cummins.md", "projects/kessler.md"} {
		if _, err := fsys.Resolve(vfs.Home, p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	projects, _ := fsys.Resolve(vfs.Home, "projects")
	if n := len(projects.Children()); n != 5 {
		t.Errorf("projects has %d files, want 5", n)
	}
}

func TestProfileLoads(t *testing.T) {
	p, err := LoadProfile()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Stack) != 5 || len(p.Links) == 0 {
		t.Fatalf("profile = %+v", p)
	}
}

// Institution names are allowed; everything else that names a place is not.
var institutions = []string{
	"Texas A&M University–Victoria",
	"University of Houston",
	"Houston City College",
}

var places = regexp.MustCompile(`(?i)\b(alabama|alaska|arizona|arkansas|california|colorado|connecticut|delaware|florida|georgia|hawaii|idaho|illinois|indiana|iowa|kansas|kentucky|louisiana|maine|maryland|massachusetts|michigan|minnesota|mississippi|missouri|montana|nebraska|nevada|new hampshire|new jersey|new mexico|new york|north carolina|north dakota|ohio|oklahoma|oregon|pennsylvania|rhode island|south carolina|south dakota|tennessee|texas|utah|vermont|virginia|washington|west virginia|wisconsin|wyoming|houston|victoria|columbus|austin|dallas|san antonio|indianapolis)\b`)

var stateCodes = regexp.MustCompile(`,\s*(AL|AK|AZ|AR|CA|CO|CT|DE|FL|GA|HI|ID|IL|IN|IA|KS|KY|LA|ME|MD|MA|MI|MN|MS|MO|MT|NE|NV|NH|NJ|NM|NY|NC|ND|OH|OK|OR|PA|RI|SC|SD|TN|TX|UT|VT|VA|WA|WV|WI|WY)\b`)

// locations returns every place name or state code in text, ignoring institution names.
func locations(text string) []string {
	for _, inst := range institutions {
		text = strings.ReplaceAll(text, inst, "")
	}
	return append(places.FindAllString(text, -1), stateCodes.FindAllString(text, -1)...)
}

func TestNoLocations(t *testing.T) {
	check := func(name, text string) {
		if found := locations(text); len(found) > 0 {
			t.Errorf("%s mentions locations: %q", name, found)
		}
	}
	err := fs.WalkDir(Files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(Files, p)
		if err != nil {
			return err
		}
		check(p, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	check("profile.yaml", string(profileYAML))
}

func TestLocationLintCatchesPlaces(t *testing.T) {
	for _, bad := range []string{"Based in Houston.", "Columbus, IN", "Worked in Texas", "Remote, TX"} {
		if len(locations(bad)) == 0 {
			t.Errorf("lint missed %q", bad)
		}
	}
	if found := locations("IT Support at University of Houston, then Texas A&M University–Victoria."); len(found) > 0 {
		t.Errorf("lint flagged institution names: %q", found)
	}
}
```

Run: `go test ./content/`
Expected: FAIL — `undefined: Files`, `undefined: LoadProfile`.

- [ ] **Step 7: Add the content package and files**

Copy is seeded from `~/DEV/Portfolio/src/content/portfolio.ts`. Keep links as bare URLs/emails (glamour prints `[text](url)` links twice). Do not add a graduation date or any location. If the Write tool is unavailable, note that `cat` is aliased to `bat` in this shell — use `command cat > file <<'EOF'` for heredocs.

Create `content/embed.go`:

```go
// Package content holds the portfolio text. The markdown files are the
// visitor's home directory; profile.yaml holds the identity strings used by
// the boot screen, whoami and neofetch.
package content

import (
	"bytes"
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Files is the home directory: every markdown file under content/.
//
//go:embed *.md work/*.md projects/*.md
var Files embed.FS

//go:embed profile.yaml
var profileYAML []byte

// Link is a labeled URL.
type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// Profile is who the portfolio is about.
type Profile struct {
	Name    string   `yaml:"name"`
	Tagline string   `yaml:"tagline"`
	Role    string   `yaml:"role"`
	School  string   `yaml:"school"`
	Degree  string   `yaml:"degree"`
	Stack   []string `yaml:"stack"`
	Links   []Link   `yaml:"links"`
}

// LoadProfile parses profile.yaml.
func LoadProfile() (Profile, error) {
	var p Profile
	dec := yaml.NewDecoder(bytes.NewReader(profileYAML))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return Profile{}, fmt.Errorf("profile.yaml: %w", err)
	}
	if p.Name == "" || p.Tagline == "" {
		return Profile{}, fmt.Errorf("profile.yaml: name and tagline are required")
	}
	return p, nil
}
```

Create `content/profile.yaml`:

```yaml
name: Kuday Yurter
tagline: CS student · data science intern at Cummins
role: Data Science Intern at Cummins
school: Texas A&M University–Victoria
degree: B.S. Computer Science
stack: [Python, SQL, Databricks, Power Platform, React]
links:
  - {label: Web, url: "https://kudayyurter.dev"}
  - {label: GitHub, url: "https://github.com/namelessmonarch0"}
  - {label: LinkedIn, url: "https://www.linkedin.com/in/kudayyurter/"}
  - {label: Email, url: "mailto:kudayyurter@gmail.com"}
```

Create `content/about.md`:

```markdown
---
title: About
summary: Who I am and where I study
order: 1
---
I’m Kuday, a Computer Science student at Texas A&M University–Victoria. I build software that replaces slow, manual work with tools people actually use.

I’m a data science intern at Cummins, where I build dashboards, a machine learning model, AI assistants, and internal apps. Before that, I ran operations for an online engraving store and taught myself Python to automate it.

## Education

- **Texas A&M University–Victoria** — B.S. Computer Science · 3.8 GPA · President’s List
- Houston City College — Associate of Science, 2022
```

Create `content/contact.md`:

```markdown
---
title: Contact
summary: Email, GitHub, LinkedIn, website
link: mailto:kudayyurter@gmail.com
order: 2
---
- Email: kudayyurter@gmail.com
- GitHub: https://github.com/namelessmonarch0
- LinkedIn: https://www.linkedin.com/in/kudayyurter/
- Website: https://kudayyurter.dev
```

Create `content/stack.md`:

```markdown
---
title: Stack
summary: Tools I use, best-known first
order: 3
---
Python · SQL · Databricks · Power Platform · React · TypeScript · JavaScript · Linux · Git · C / C++ · Power BI · scikit-learn · Copilot Studio · AWS · Azure · Docker · Neovim · Rust · C# · .NET · Node.js · FastAPI · Unreal Engine · Unity · MATLAB
```

Create `content/work/cummins.md`:

```markdown
---
title: Cummins
summary: Data Science Intern
date: May 2026 — Present
order: 1
---
## Highlights

- Built dashboards and a machine learning model for the turbo balancing line, which helped raise output from 35 to 55 turbos a day per machine.
- Built AI assistants for patent review and project intake, an app that replaced 5+ skills trackers, and the department’s SharePoint site.

## Projects

### Turbo Balancer dashboards

Live dashboards for Cummins’ turbo balancing machines, replacing hand-entered data and Excel reports. Engineers used them to find the slow spots on the line.

- **Result:** 35 → 55 turbos a day per machine
- **Built with:** Databricks · Python · Power BI

### Balancer correction model

A model that reproduces the correction the balancing machine calculates, matching it about 98% of the time, so the machines Cummins already owns can do more.

- **Result:** About $16M in new machines not needed
- **Built with:** Python · scikit-learn

### Engineering AI agents

Assistants that compare new patents against Cummins’ own, turn rough project ideas into proposals, and help with quality decisions — all answering from internal documents.

- **Result:** Patent reviews take half the time
- **Built with:** Copilot Studio · AWS Bedrock · Databricks

### Skills & Capabilities app

One app for tracking who knows what, replacing 5+ scattered spreadsheets and tools. Rolled out first to a 30-person engineering team.

- **Result:** About 60% less time spent managing skills
- **Built with:** React · Power Apps · Dataverse

### CCS AI SharePoint site

The home for AI work across Cummins’ components and software group: news, a list of live and in-progress agents, and training.

- **Result:** Live for a ~20,000-person organization
- **Built with:** SharePoint
```

Create `content/work/engrave-me-now.md`:

```markdown
---
title: Engrave Me Now
summary: Operations Manager
date: Jan 2025 — May 2026
order: 2
---
## Highlights

- Ran the production floor on my own for a store shipping 1,000+ orders a week, with a 4.9/5 rating across three storefronts.
- Built the sales and inventory system the business ran on, and kept four laser and UV machines running 99% of the time.

## Projects

### Store sales & inventory system

Pulls orders and stock from Amazon, Etsy, and a third store into one place, forecasts demand, and ranks what to restock before each order.

- **Result:** Inventory costs down 20%, sales up 30%
- **Built with:** Python · Amazon and Etsy APIs
```

Create `content/work/university-of-houston.md`:

```markdown
---
title: University of Houston
summary: IT Support Specialist
date: Jan 2023 — Aug 2024
order: 3
---
## Highlights

- Fixed hardware and software problems for faculty and students in the College of Technology, with a 95% satisfaction rate.
- Kept 20+ computer labs up to date and set up new labs from unboxed hardware to networked machines.
```

Create `content/work/ifixandrepair.md`:

```markdown
---
title: IFixandRepair
summary: Store Manager & Repair Technician
date: Dec 2020 — May 2022
order: 4
---
## Highlights

- Repaired 100+ phones, tablets, and laptops — screens, cameras, back glass, and system recovery — with a 95% success rate.
- Ran the store alone on many shifts, from customer intake to repairs to closing.
```

Create `content/projects/kessler.md`:

```markdown
---
title: Kessler
summary: Live 3D globe of everything tracked in Earth orbit
stack: Next.js · Three.js · FastAPI · AWS
link: https://kessler.kudayyurter.dev
order: 1
---
Every tracked object in Earth orbit, 1957 to now: a live 3D globe of about 30,000 objects at their real positions, plus charts of how orbit got crowded. It started as my team’s MATLAB app that took 1st place out of 25 teams at the Grand Challenge Winter Summit.
```

Create `content/projects/dispatch.md`:

```markdown
---
title: Dispatch
summary: Several coding agents side by side in one terminal
stack: Rust
link: https://github.com/namelessmonarch0/Dispatch
order: 2
---
A terminal app for running several coding agents (Claude Code, Codex, opencode) side by side in live, tiled terminals. Early stage.
```

Create `content/projects/snake-game.md`:

```markdown
---
title: Snake Game
summary: A retro snake game
stack: C++ · Raylib
link: https://github.com/namelessmonarch0/SnakeGame
order: 3
---
A retro snake game.
```

Create `content/projects/clash-of-valor.md`:

```markdown
---
title: Clash of Valor
summary: A duel game that runs in the terminal
stack: C++
link: https://github.com/namelessmonarch0/ClashOfValor
order: 4
---
A duel game that runs in the terminal.
```

Create `content/projects/lumon-boot-splash.md`:

```markdown
---
title: Lumon boot splash
summary: Linux boot animation styled after Lumon from Severance
stack: Shell · Plymouth
link: https://github.com/namelessmonarch0/PlymouthLumonSplash
order: 5
---
A Linux boot animation styled after Lumon, the company in Severance.
```

- [ ] **Step 8: Run the content tests**

Run: `go test ./content/ ./internal/vfs/ -v`
Expected: PASS, including `TestNoLocations` and `TestLocationLintCatchesPlaces`.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum internal/vfs content
git commit -m "feat: add virtual filesystem and portfolio content

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 3: Style and markdown rendering

**Files:**
- Create: `internal/style/style.go`, `internal/style/markdown.go`, `internal/style/theme.json`
- Test: `internal/style/style_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `var style.Muted, style.Faint, style.Rule, style.Bold lipgloss.Style`
  - `func style.Link(url, text string) string` (OSC 8 hyperlink); `func style.LinkLabel(url string) string`
  - `func style.Markdown(md string, width int) (string, error)` — width <20 → 80, >100 → 100; no trailing padding; continuation lines of bullets indented

- [ ] **Step 1: Add dependencies**

```bash
go get charm.land/lipgloss/v2@v2.0.6 charm.land/glamour/v2@v2.0.1 github.com/charmbracelet/x/ansi
```

- [ ] **Step 2: Write the failing test**

Create `internal/style/style_test.go`:

```go
package style

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMarkdownRendersPlainText(t *testing.T) {
	out, err := Markdown("## Highlights\n\n- Built **dashboards**.\n\nSee [site](https://example.com).\n", 60)
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(out)
	for _, want := range []string{"Highlights", "• Built dashboards.", "site", "https://example.com"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in:\n%s", want, plain)
		}
	}
	for _, l := range strings.Split(plain, "\n") {
		if strings.HasSuffix(l, " ") {
			t.Errorf("line has trailing padding: %q", l)
		}
	}
	if strings.HasPrefix(plain, "\n") || strings.HasSuffix(plain, "\n") {
		t.Errorf("output not trimmed: %q", plain)
	}
}

// Review focus 1: light-background terminals.
func TestMarkdownNeverForcesAForegroundOnText(t *testing.T) {
	out, err := Markdown("# Title\n\nJust words and **bold** words.\n", 60)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "38;2;255;255;255") || strings.Contains(out, "\x1b[38") {
		t.Fatalf("text has a hard-coded foreground color: %q", out)
	}
}

// Review focus 3: unknown or tiny widths.
func TestMarkdownWidthFallbacks(t *testing.T) {
	long := strings.Repeat("word ", 60)
	for _, w := range []int{0, -5, 5} {
		out, err := Markdown(long, w)
		if err != nil {
			t.Fatalf("width %d: %v", w, err)
		}
		for _, l := range strings.Split(ansi.Strip(out), "\n") {
			if ansi.StringWidth(l) > 80 {
				t.Fatalf("width %d: line wider than 80: %q", w, l)
			}
		}
	}
	out, _ := Markdown(long, 300)
	for _, l := range strings.Split(ansi.Strip(out), "\n") {
		if ansi.StringWidth(l) > 100 {
			t.Fatalf("wide terminal: line wider than 100: %q", l)
		}
	}
}

func TestLinkLabel(t *testing.T) {
	cases := map[string]string{
		"https://www.linkedin.com/in/kudayyurter/": "linkedin.com/in/kudayyurter",
		"mailto:kudayyurter@gmail.com":             "kudayyurter@gmail.com",
		"https://kudayyurter.dev":                  "kudayyurter.dev",
	}
	for in, want := range cases {
		if got := LinkLabel(in); got != want {
			t.Errorf("LinkLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinkIsOSC8(t *testing.T) {
	got := Link("https://example.com", "example")
	if !strings.Contains(got, "\x1b]8;;https://example.com") || ansi.Strip(got) != "example" {
		t.Fatalf("Link = %q", got)
	}
}
```

- [ ] **Step 3: Run it to verify it fails**

Run: `go test ./internal/style/`
Expected: FAIL — `undefined: Markdown`, `undefined: Link`, `undefined: LinkLabel`.

- [ ] **Step 4: Implement**

The theme sets **no** text or heading color (Review Focus 1). Glamour's own word wrap breaks inside tokens like `4.9/5`, so it is disabled (`WithWordWrap(0)`) and lines are wrapped with `ansi.Wrap`, which only breaks at spaces.

Create `internal/style/theme.json`:

```json
{
  "document": { "margin": 0 },
  "paragraph": {},
  "heading": { "bold": true, "block_suffix": "\n" },
  "h1": { "prefix": "" },
  "h2": { "prefix": "" },
  "h3": { "prefix": "" },
  "strong": { "bold": true },
  "emph": { "italic": true },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "list": { "level_indent": 2 },
  "link": { "color": "#a3a3a3", "underline": true },
  "link_text": { "bold": true },
  "hr": { "color": "#262626", "format": "\n────────\n" },
  "code": { "color": "#a3a3a3" },
  "code_block": { "color": "#a3a3a3", "margin": 2 }
}
```

Create `internal/style/style.go`:

```go
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
```

Create `internal/style/markdown.go`:

```go
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
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/style/ -v`
Expected: PASS (5 tests).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/style
git commit -m "feat: add monochrome style and markdown rendering

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 4: Shell core and navigation commands

**Files:**
- Create: `internal/shell/session.go`, `internal/shell/parse.go`, `internal/shell/suggest.go`, `internal/shell/nav.go`
- Test: `internal/shell/helpers_test.go`, `internal/shell/nav_test.go`

**Interfaces:**
- Consumes: `vfs.FS`, `vfs.Node`, `vfs.Home`, `vfs.Display`, `vfs.ErrNotDir` (Task 2); `content.Profile` (Task 2); `style.Muted/Faint/Rule/Bold` (Task 3).
- Produces:
  - `type shell.Action int` with `ActionNone, ActionClear, ActionExit, ActionBoot`
  - `type shell.Result struct { Output string; Code int; Action Action }`
  - `type shell.Options struct { Width int; Interactive bool; Now func() time.Time }`
  - `func shell.New(fsys *vfs.FS, profile content.Profile, o Options) *Session`
  - `func (s *Session) Run(line string) Result`, `Prompt() string`, `Cwd() string`, `History() []string`, `SetWidth(w int)`
  - Package-internal for Tasks 5–6: `type command struct { run func(*Session, []string) Result; group, usage, about string }`, `register(name string, c command)`, `visibleCommands() []string`, `ok(string) Result`, `fail(string) Result`, `pathErr(cmd, p string, err error) Result`, `parseFlags(cmd string, args []string, allowed string) (map[rune]bool, []string, error)`, `entry`, `displayName(entry) string`
  - Commands registered here: `ls`, `cd`, `pwd`, `tree`

- [ ] **Step 1: Write the test helpers and failing tests**

The fixture tree is fixed so tests don't change when the real content does.

Create `internal/shell/helpers_test.go`:

```go
package shell

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

// newTestSession uses a small fixed tree so tests don't break when the real
// portfolio content changes.
func newTestSession(t *testing.T) *Session {
	t.Helper()
	fsys, err := vfs.New(fstest.MapFS{
		"about.md":            {Data: []byte("---\ntitle: About\nsummary: Who I am\norder: 1\n---\nHello **there**.\n")},
		"contact.md":          {Data: []byte("---\ntitle: Contact\nsummary: How to reach me\nlink: mailto:me@example.com\norder: 2\n---\nEmail me.\n")},
		"work/acme.md":        {Data: []byte("---\ntitle: Acme\nsummary: Engineer\ndate: 2025 — Present\norder: 1\n---\nBuilt things.\n")},
		"work/beta.md":        {Data: []byte("---\ntitle: Beta\nsummary: Intern\ndate: 2024\norder: 2\n---\nLearned things.\n")},
		"projects/kessler.md": {Data: []byte("---\ntitle: Kessler\nsummary: Orbit globe\nstack: Go · Three.js\nlink: https://kessler.example.com\n---\nA globe.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := content.Profile{
		Name:    "Test Person",
		Tagline: "tester of things",
		Role:    "Tester at Acme",
		School:  "Test University",
		Degree:  "B.S. Testing",
		Stack:   []string{"Go", "SQL"},
		Links:   []content.Link{{Label: "Web", URL: "https://example.com"}},
	}
	return New(fsys, profile, Options{
		Width:       80,
		Interactive: true,
		Now:         func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) },
	})
}

// plain strips styling and trailing spaces so tests compare visible text.
func plain(r Result) string {
	lines := strings.Split(ansi.Strip(r.Output), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func run(t *testing.T, s *Session, line string) (string, Result) {
	t.Helper()
	r := s.Run(line)
	return plain(r), r
}
```

Create `internal/shell/nav_test.go`:

```go
package shell

import (
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	s := newTestSession(t)
	cases := []struct{ line, want string }{
		{"ls", "projects/  work/  about.md  contact.md"},
		{"ls work", "acme.md  beta.md"},
		{"ls -a work", "./  ../  acme.md  beta.md"},
		{"ls about.md", "about.md"},
		{"ls /", "home/"},
		{"ls -l", strings.Join([]string{
			"drwxr-xr-x  projects/   1 item",
			"drwxr-xr-x  work/       2 items",
			"-rw-r--r--  about.md    Who I am",
			"-rw-r--r--  contact.md  How to reach me",
		}, "\n")},
		{"ls -la work", strings.Join([]string{
			"drwxr-xr-x  ./",
			"drwxr-xr-x  ../",
			"-rw-r--r--  acme.md  Engineer · 2025 — Present",
			"-rw-r--r--  beta.md  Intern · 2024",
		}, "\n")},
		{"ls work projects", "work:\nacme.md  beta.md\n\nprojects:\nkessler.md"},
	}
	for _, c := range cases {
		got, r := run(t, s, c.line)
		if got != c.want || r.Code != 0 {
			t.Errorf("%q (code %d) =\n%s\nwant\n%s", c.line, r.Code, got, c.want)
		}
	}
}

func TestLsWrapsToWidth(t *testing.T) {
	s := newTestSession(t)
	s.SetWidth(20)
	got, _ := run(t, s, "ls")
	for _, l := range strings.Split(got, "\n") {
		if len(l) > 20 {
			t.Fatalf("line wider than 20: %q", l)
		}
	}
}

func TestLsErrors(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "ls nope"); got != "ls: nope: No such file or directory" || r.Code != 1 {
		t.Errorf("ls nope = %q (%d)", got, r.Code)
	}
	if got, r := run(t, s, "ls -z"); got != "ls: invalid option -- 'z'" || r.Code != 1 {
		t.Errorf("ls -z = %q (%d)", got, r.Code)
	}
}

func TestCdAndPwd(t *testing.T) {
	s := newTestSession(t)
	steps := []struct{ line, out, cwd string }{
		{"cd work", "", "/home/guest/work"},
		{"pwd", "/home/guest/work", "/home/guest/work"},
		{"cd ..", "", "/home/guest"},
		{"cd -", "~/work", "/home/guest/work"},
		{"cd", "", "/home/guest"},
		{"cd /", "", "/"},
		{"cd ~/projects", "", "/home/guest/projects"},
		{"cd ../../../../..", "", "/"},
		{"cd /home", "", "/home"},
	}
	for _, st := range steps {
		got, r := run(t, s, st.line)
		if got != st.out || s.Cwd() != st.cwd || r.Code != 0 {
			t.Errorf("%q: out %q cwd %q code %d; want out %q cwd %q", st.line, got, s.Cwd(), r.Code, st.out, st.cwd)
		}
	}
}

func TestCdErrors(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"cd nope":     "cd: nope: No such file or directory",
		"cd about.md": "cd: about.md: Not a directory",
		"cd a b":      "cd: too many arguments",
		"cd -":        "cd: OLDPWD not set",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
	if s.Cwd() != "/home/guest" {
		t.Fatalf("failed cd changed cwd to %q", s.Cwd())
	}
}

func TestPromptFollowsCwd(t *testing.T) {
	s := newTestSession(t)
	if got := plain(Result{Output: s.Prompt()}); got != "guest@kuday:~$" {
		t.Fatalf("prompt = %q", got)
	}
	s.Run("cd work")
	if got := plain(Result{Output: s.Prompt()}); got != "guest@kuday:~/work$" {
		t.Fatalf("prompt = %q", got)
	}
}

func TestTree(t *testing.T) {
	s := newTestSession(t)
	want := strings.Join([]string{
		".",
		"├── projects/",
		"│   └── kessler.md",
		"├── work/",
		"│   ├── acme.md",
		"│   └── beta.md",
		"├── about.md",
		"└── contact.md",
		"",
		"2 directories, 5 files",
	}, "\n")
	if got, _ := run(t, s, "tree"); got != want {
		t.Fatalf("tree =\n%s\nwant\n%s", got, want)
	}
	if got, _ := run(t, s, "tree work"); !strings.HasPrefix(got, "work\n├── acme.md") {
		t.Fatalf("tree work =\n%s", got)
	}
}

func TestNotFoundAndSuggestions(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"sl":     "sl: command not found\ndid you mean ls?",
		"pdw":    "pdw: command not found\ndid you mean pwd?",
		"x":      "x: command not found",
		"banana": "banana: command not found",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 127 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestParsing(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "ls | grep x"); got != "pipes and redirects aren't supported here" || r.Code != 1 {
		t.Errorf("pipe = %q (%d)", got, r.Code)
	}
	if got, _ := run(t, s, "ls > out"); got != "pipes and redirects aren't supported here" {
		t.Errorf("redirect = %q", got)
	}
	if got, _ := run(t, s, `ls "work`); got != "unexpected EOF while looking for matching `\"'" {
		t.Errorf("unterminated = %q", got)
	}
	if got, _ := run(t, s, `ls "work"`); got != "acme.md  beta.md" {
		t.Errorf("quoted = %q", got)
	}
	if r := s.Run("   "); r.Output != "" || r.Code != 0 {
		t.Errorf("blank line = %+v", r)
	}
}

func TestHistoryRecordsNonBlankLines(t *testing.T) {
	s := newTestSession(t)
	s.Run("ls")
	s.Run("")
	s.Run("nope")
	if h := s.History(); len(h) != 2 || h[0] != "ls" || h[1] != "nope" {
		t.Fatalf("history = %q", h)
	}
}

// Review focus 2: control characters never reach the output.
func TestControlCharactersAreStripped(t *testing.T) {
	s := newTestSession(t)
	r := s.Run("ls \x1b[2J\x1b]0;pwned\x07work")
	if strings.Contains(r.Output, "\x1b[2J") || strings.Contains(r.Output, "\x1b]0;") {
		t.Fatalf("escape sequence leaked: %q", r.Output)
	}
	r = s.Run("\x1b[31mfoo")
	if strings.Contains(r.Output, "\x1b[31m") {
		t.Fatalf("escape sequence leaked: %q", r.Output)
	}
}

// Review focus 4: nothing outside the virtual tree is reachable.
func TestPathEscapes(t *testing.T) {
	s := newTestSession(t)
	for _, p := range []string{"../../../../etc/passwd", "/etc", "/home/guest/../../root", "~/../../proc"} {
		if _, r := run(t, s, "ls "+p); r.Code != 1 {
			t.Errorf("ls %s succeeded", p)
		}
	}
}
```

`TestControlCharactersAreStripped` pins Review Focus 2 and `TestPathEscapes` pins Review Focus 4.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/shell/`
Expected: FAIL — `undefined: New`, `undefined: Session`, `undefined: Result`.

- [ ] **Step 3: Implement the session and parser**

Create `internal/shell/session.go`:

```go
// Package shell is the pretend shell: it turns a command line into styled
// output. It knows nothing about terminals or SSH, so the same code serves
// interactive sessions and one-shot `ssh host <command>` runs.
package shell

import (
	"sort"
	"strings"
	"time"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

// Action is something the caller must do beyond printing output.
type Action int

const (
	ActionNone  Action = iota
	ActionClear        // clear the screen
	ActionExit         // end the session
	ActionBoot         // replay the boot animation
)

// Result is what running one line produced.
type Result struct {
	Output string // styled; may be empty
	Code   int    // 0 ok, 1 error, 127 command not found
	Action Action
}

// Options configure a Session.
type Options struct {
	Width       int              // terminal columns; 0 means unknown
	Interactive bool             // false for `ssh host <command>`
	Now         func() time.Time // nil means time.Now
}

// Session is one visitor's shell state.
type Session struct {
	fs          *vfs.FS
	profile     content.Profile
	cwd, prev   string
	history     []string
	width       int
	interactive bool
	now         func() time.Time
}

// New starts a session in the home directory.
func New(fsys *vfs.FS, profile content.Profile, o Options) *Session {
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Session{fs: fsys, profile: profile, cwd: vfs.Home, width: o.Width, interactive: o.Interactive, now: o.Now}
}

// SetWidth updates the terminal width used to wrap output.
func (s *Session) SetWidth(w int) { s.width = w }

// Cwd is the absolute working directory.
func (s *Session) Cwd() string { return s.cwd }

// History is every non-blank line run so far, oldest first.
func (s *Session) History() []string { return s.history }

// Prompt is the styled prompt, e.g. "guest@kuday:~/work$ ".
func (s *Session) Prompt() string {
	return style.Muted.Render("guest@kuday") + ":" + style.Bold.Render(vfs.Display(s.cwd)) + "$ "
}

// Run executes one command line.
func (s *Session) Run(line string) Result {
	line = sanitize(line)
	if strings.TrimSpace(line) == "" {
		return Result{}
	}
	s.history = append(s.history, line)
	words, err := split(line)
	if err != nil {
		return fail(err.Error())
	}
	name, args := words[0], words[1:]
	c, ok := commands[name]
	if !ok {
		return notFound(name)
	}
	return c.run(s, args)
}

// command is one entry in the registry.
type command struct {
	run   func(s *Session, args []string) Result
	group string // help section; "" hides it from help, completion and suggestions
	usage string // e.g. "ls [-la] [path]"
	about string // one line for help
}

var commands = map[string]command{}

func register(name string, c command) { commands[name] = c }

// visibleCommands are the names shown in help and offered by completion.
func visibleCommands() []string {
	var names []string
	for name, c := range commands {
		if c.group != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func ok(out string) Result   { return Result{Output: out} }
func fail(out string) Result { return Result{Output: out, Code: 1} }
```

Create `internal/shell/parse.go`:

```go
package shell

import (
	"errors"
	"fmt"
	"strings"
)

var errUnsupported = errors.New("pipes and redirects aren't supported here")

// sanitize drops control characters so nothing a visitor types can smuggle
// terminal escape sequences into output. Tabs become spaces.
func sanitize(line string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, line)
}

// split turns a command line into words, honoring single and double quotes.
// Shell operators are rejected because nothing here can pipe or redirect.
func split(line string) ([]string, error) {
	var (
		words  []string
		cur    strings.Builder
		inWord bool
		quote  rune
	)
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		case strings.ContainsRune("|<>;&", r):
			return nil, errUnsupported
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unexpected EOF while looking for matching `%c'", quote)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// parseFlags separates single-letter flags (which may be combined, like -la)
// from operands. Letters not in allowed are an error in GNU wording.
func parseFlags(cmd string, args []string, allowed string) (map[rune]bool, []string, error) {
	set := map[rune]bool{}
	var operands []string
	for _, a := range args {
		if len(a) > 1 && a[0] == '-' {
			for _, r := range a[1:] {
				if !strings.ContainsRune(allowed, r) {
					return nil, nil, fmt.Errorf("%s: invalid option -- '%c'", cmd, r)
				}
				set[r] = true
			}
			continue
		}
		operands = append(operands, a)
	}
	return set, operands, nil
}
```

Create `internal/shell/suggest.go`:

```go
package shell

import "github.com/namelessmonarch0/ssh-portfolio/internal/style"

func notFound(name string) Result {
	out := name + ": command not found"
	if m := closest(name); m != "" {
		out += "\n" + style.Faint.Render("did you mean "+m+"?")
	}
	return Result{Output: out, Code: 127}
}

// closest is the visible command nearest to name, if it is close enough to
// be a typo: at most 2 edits, and at most half of a short name.
func closest(name string) string {
	best, bestD := "", 3
	for _, c := range visibleCommands() {
		d := distance(name, c)
		if d < bestD {
			best, bestD = c, d
		}
	}
	if best == "" || bestD > len([]rune(name))/2+1 {
		return ""
	}
	return best
}

// distance is the Damerau–Levenshtein distance (swapped neighbors count as
// one edit, so "sl" is one step from "ls").
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range rb {
		d[0][j+1] = j + 1
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}
```

- [ ] **Step 4: Implement the navigation commands**

Create `internal/shell/nav.go`:

```go
package shell

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
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
		return style.Bold.Render(e.name + "/")
	}
	return e.name
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
			blocks = append(blocks, fmt.Sprintf("ls: %s: %s", p, err))
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
	b.WriteString(style.Bold.Render(label))
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
```

- [ ] **Step 5: Run the tests**

Run: `go vet ./internal/shell/ && go test ./internal/shell/ -v`
Expected: PASS (TestLs, TestLsWrapsToWidth, TestLsErrors, TestCdAndPwd, TestCdErrors, TestPromptFollowsCwd, TestTree, TestNotFoundAndSuggestions, TestParsing, TestHistoryRecordsNonBlankLines, TestControlCharactersAreStripped, TestPathEscapes).

- [ ] **Step 6: Commit**

```bash
git add internal/shell
git commit -m "feat: add shell core with ls, cd, pwd and tree

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 5: Reading, about, and utility commands

**Files:**
- Create: `internal/shell/read.go`, `internal/shell/about.go`, `internal/shell/misc.go`
- Test: `internal/shell/commands_test.go`

**Interfaces:**
- Consumes: everything package-internal from Task 4; `style.Markdown`, `style.Link`, `style.LinkLabel` (Task 3); `pixelfont.HalfBlocks/Scale/MustText` (Task 1).
- Produces: commands `cat` (`less`, `more` hidden aliases), `open`, `whoami`, `neofetch`, `help`, `clear`, `history`, `echo`, `date`, `exit` (`logout` hidden), `boot`, and hidden easter eggs `sudo`, `rm`, `mv`, `cp`, `touch`, `mkdir`, `vim`, `vi`, `nano`, `emacs`. Help groups, in order: `Moving around`, `Reading`, `About`, `Utilities`.

- [ ] **Step 1: Write the failing tests**

Create `internal/shell/commands_test.go`:

```go
package shell

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func TestCat(t *testing.T) {
	s := newTestSession(t)
	got, r := run(t, s, "cat about.md")
	if got != "About\n\nHello there." || r.Code != 0 {
		t.Errorf("cat about.md = %q (%d)", got, r.Code)
	}
	got, _ = run(t, s, "cat projects/kessler.md")
	want := "Kessler\nGo · Three.js\nkessler.example.com\n\nA globe."
	if got != want {
		t.Errorf("cat kessler =\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(s.Run("cat projects/kessler.md").Output, "\x1b]8;;https://kessler.example.com") {
		t.Error("link is not an OSC 8 hyperlink")
	}
	got, _ = run(t, s, "cat work/acme.md")
	if !strings.HasPrefix(got, "Acme\n2025 — Present\n\n") {
		t.Errorf("cat acme = %q", got)
	}
}

func TestCatErrorsAndAliases(t *testing.T) {
	s := newTestSession(t)
	cases := []struct {
		line, want string
		code       int
	}{
		{"cat", "cat: missing file operand", 1},
		{"cat work", "cat: work: Is a directory", 1},
		{"cat nope", "cat: nope: No such file or directory", 1},
		{"cat about.md/", "cat: about.md/: Not a directory", 1},
		{"cat about.md nope", "About\n\nHello there.\n\ncat: nope: No such file or directory", 1},
		{"less about.md", "About\n\nHello there.", 0},
		{"more work", "more: work: Is a directory", 1},
		{"cat -n about.md", "cat: invalid option -- 'n'", 1},
	}
	for _, c := range cases {
		if got, r := run(t, s, c.line); got != c.want || r.Code != c.code {
			t.Errorf("%q = %q (%d), want %q (%d)", c.line, got, r.Code, c.want, c.code)
		}
	}
}

func TestOpen(t *testing.T) {
	s := newTestSession(t)
	if got, r := run(t, s, "open projects/kessler.md"); got != "https://kessler.example.com" || r.Code != 0 {
		t.Errorf("open = %q (%d)", got, r.Code)
	}
	cases := map[string]string{
		"open about.md": "open: about.md: no link",
		"open work":     "open: work: no link",
		"open nope":     "open: nope: No such file or directory",
		"open":          "usage: open <file>",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestWhoami(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, "whoami"); got != "guest — visiting Test Person · tester of things" {
		t.Errorf("whoami = %q", got)
	}
}

func TestNeofetch(t *testing.T) {
	s := newTestSession(t)
	got, _ := run(t, s, "neofetch")
	for _, want := range []string{"guest@kuday", "Tester at Acme", "Go · SQL", "example.com", "██      ██  ██      ██"} {
		if !strings.Contains(got, want) {
			t.Errorf("neofetch missing %q:\n%s", want, got)
		}
	}
	for _, l := range strings.Split(s.Run("neofetch").Output, "\n") {
		if strings.HasSuffix(l, " ") {
			t.Errorf("neofetch line has trailing padding: %q", l)
		}
	}
	s.SetWidth(50)
	got, _ = run(t, s, "neofetch")
	if strings.Contains(got, "██") || !strings.Contains(got, "Test University") {
		t.Errorf("narrow neofetch should drop the logo:\n%s", got)
	}
}

func TestHelpListsVisibleCommandsOnly(t *testing.T) {
	s := newTestSession(t)
	got, _ := run(t, s, "help")
	for _, want := range []string{"Moving around", "Reading", "About", "Utilities", "ls [-la] [path]", "cat <file>", "neofetch", "boot"} {
		if !strings.Contains(got, want) {
			t.Errorf("help missing %q", want)
		}
	}
	for _, hidden := range []string{"sudo", "vim", "rm", "less", "logout"} {
		if strings.Contains(got, "  "+hidden+" ") || strings.Contains(got, "\n  "+hidden) {
			t.Errorf("help shows hidden command %q", hidden)
		}
	}
}

func TestUtilities(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, `echo hello   "big world"`); got != "hello big world" {
		t.Errorf("echo = %q", got)
	}
	if got, _ := run(t, s, "date"); got != "Mon Sep 28 12:00:00 UTC 2026" {
		t.Errorf("date = %q", got)
	}
	if got, _ := run(t, s, "history"); got != "    1  echo hello   \"big world\"\n    2  date\n    3  history" {
		t.Errorf("history = %q", got)
	}
	if r := s.Run("clear"); r.Action != ActionClear || r.Output != "" {
		t.Errorf("clear = %+v", r)
	}
	for _, line := range []string{"exit", "logout"} {
		if r := s.Run(line); r.Action != ActionExit || r.Output != "logout" {
			t.Errorf("%s = %+v", line, r)
		}
	}
	if r := s.Run("boot"); r.Action != ActionBoot {
		t.Errorf("boot = %+v", r)
	}
}

func TestBootNeedsATerminal(t *testing.T) {
	fsys, err := vfs.New(fstest.MapFS{"a.md": {Data: []byte("---\ntitle: A\nsummary: B\n---\n")}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(fsys, content.Profile{}, Options{})
	if r := s.Run("boot"); r.Code != 1 || r.Action != ActionNone {
		t.Errorf("non-interactive boot = %+v", r)
	}
}

func TestEasterEggs(t *testing.T) {
	s := newTestSession(t)
	cases := map[string]string{
		"sudo rm -rf /": "guest is not in the sudoers file. This incident will be reported.",
		"rm about.md":   "rm: read-only file system (nice try)",
		"mkdir x":       "mkdir: read-only file system (nice try)",
		"vim about.md":  "vim: read-only file system — try cat about.md",
		"nano":          "nano: read-only file system — try cat <file>",
	}
	for line, want := range cases {
		if got, r := run(t, s, line); got != want || r.Code != 1 {
			t.Errorf("%q = %q (%d), want %q", line, got, r.Code, want)
		}
	}
}

func TestSuggestionsIncludeLaterCommands(t *testing.T) {
	s := newTestSession(t)
	if got, _ := run(t, s, "cta"); got != "cta: command not found\ndid you mean cat?" {
		t.Errorf("cta = %q", got)
	}
	if got, _ := run(t, s, "hlep"); got != "hlep: command not found\ndid you mean help?" {
		t.Errorf("hlep = %q", got)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/shell/ -run 'Cat|Open|Whoami|Neofetch|Help|Utilities|Boot|Easter|Suggestions'`
Expected: FAIL — e.g. `"cat about.md" = "cat: command not found"`, `"cta" = "cta: command not found"` (no `did you mean cat?` yet).

- [ ] **Step 3: Implement**

Create `internal/shell/read.go`:

```go
package shell

import (
	"errors"
	"fmt"
	"strings"

	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

var errIsDir = errors.New("Is a directory")

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
		var parts []string
		code := 0
		for _, p := range ops {
			n, err := s.fs.Resolve(s.cwd, p)
			if err == nil && n.Dir {
				err = errIsDir
			}
			if err != nil {
				parts = append(parts, fmt.Sprintf("%s: %s: %s", name, p, err))
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
	b.WriteString(style.Bold.Render(n.Meta.Title))
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
		b.WriteString("\n" + style.Link(n.Meta.Link, style.Muted.Render(style.LinkLabel(n.Meta.Link))))
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
```

Create `internal/shell/about.go`:

```go
package shell

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/namelessmonarch0/ssh-portfolio/internal/pixelfont"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
)

func init() {
	register("whoami", command{run: runWhoami, group: "About", usage: "whoami", about: "who you are, and whose portfolio this is"})
	register("neofetch", command{run: runNeofetch, group: "About", usage: "neofetch", about: "the whole portfolio on one card"})
	register("help", command{run: runHelp, group: "Utilities", usage: "help", about: "this list"})
}

func runWhoami(s *Session, _ []string) Result {
	return ok("guest " + style.Muted.Render("— visiting "+s.profile.Name+" · "+s.profile.Tagline))
}

// monogram is "KY" in the site's pixel font at 2×: 22 columns × 7 rows.
var monogram = pixelfont.HalfBlocks(pixelfont.Scale(pixelfont.MustText("KY"), 2))

func runNeofetch(s *Session, _ []string) Result {
	p := s.profile
	rows := [][2]string{
		{"Name", p.Name},
		{"Role", p.Role},
		{"School", p.School},
		{"Degree", p.Degree},
		{"Stack", strings.Join(p.Stack, " · ")},
	}
	for _, l := range p.Links {
		rows = append(rows, [2]string{l.Label, style.Link(l.URL, style.LinkLabel(l.URL))})
	}
	info := []string{style.Bold.Render("guest@kuday"), style.Rule.Render(strings.Repeat("─", 11))}
	for _, r := range rows {
		if r[1] != "" {
			info = append(info, style.Muted.Render(fmt.Sprintf("%-9s", r[0]))+r[1])
		}
	}
	card := strings.Join(info, "\n")
	if s.width >= 72 || s.width == 0 {
		card = lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(monogram, "\n"), "   ", card)
	}
	lines := strings.Split(card, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return ok(strings.Join(lines, "\n"))
}

var helpGroups = []string{"Moving around", "Reading", "About", "Utilities"}

func runHelp(_ *Session, _ []string) Result {
	var b strings.Builder
	for i, g := range helpGroups {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(style.Bold.Render(g))
		var names []string
		for name, c := range commands {
			if c.group == g {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			c := commands[name]
			fmt.Fprintf(&b, "\n  %-18s%s", c.usage, style.Muted.Render(c.about))
		}
	}
	b.WriteString("\n\n" + style.Faint.Render("Tab completes · ↑/↓ history · Ctrl-L clears · exit leaves"))
	return ok(b.String())
}
```

Create `internal/shell/misc.go`:

```go
package shell

import (
	"fmt"
	"strings"
)

func init() {
	register("clear", command{run: runClear, group: "Utilities", usage: "clear", about: "clear the screen"})
	register("history", command{run: runHistory, group: "Utilities", usage: "history", about: "commands you've run"})
	register("echo", command{run: runEcho, group: "Utilities", usage: "echo [text]", about: "print text"})
	register("date", command{run: runDate, group: "Utilities", usage: "date", about: "current time (UTC)"})
	register("exit", command{run: runExit, group: "Utilities", usage: "exit", about: "leave"})
	register("logout", command{run: runExit})
	register("boot", command{run: runBoot, group: "Utilities", usage: "boot", about: "replay the startup animation"})

	register("sudo", command{run: func(*Session, []string) Result {
		return fail("guest is not in the sudoers file. This incident will be reported.")
	}})
	for _, name := range []string{"rm", "mv", "cp", "touch", "mkdir"} {
		register(name, command{run: readOnly(name)})
	}
	for _, name := range []string{"vim", "vi", "nano", "emacs"} {
		register(name, command{run: editor(name)})
	}
}

func runClear(*Session, []string) Result { return Result{Action: ActionClear} }

func runHistory(s *Session, _ []string) Result {
	lines := make([]string, len(s.history))
	for i, h := range s.history {
		lines[i] = fmt.Sprintf("%5d  %s", i+1, h)
	}
	return ok(strings.Join(lines, "\n"))
}

func runEcho(_ *Session, args []string) Result { return ok(strings.Join(args, " ")) }

func runDate(s *Session, _ []string) Result {
	return ok(s.now().UTC().Format("Mon Jan _2 15:04:05 UTC 2006"))
}

func runExit(*Session, []string) Result { return Result{Output: "logout", Action: ActionExit} }

func runBoot(s *Session, _ []string) Result {
	if !s.interactive {
		return fail("boot: needs an interactive terminal (try ssh -t)")
	}
	return Result{Action: ActionBoot}
}

func readOnly(name string) func(*Session, []string) Result {
	return func(*Session, []string) Result {
		return fail(name + ": read-only file system (nice try)")
	}
}

func editor(name string) func(*Session, []string) Result {
	return func(_ *Session, args []string) Result {
		target := "<file>"
		if len(args) > 0 {
			target = args[0]
		}
		return fail(name + ": read-only file system — try cat " + target)
	}
}
```

- [ ] **Step 4: Run all shell tests**

Run: `go vet ./internal/shell/ && go test ./internal/shell/ -v`
Expected: PASS, including every Task 4 test (`banana` must still have no suggestion now that `help` exists).

- [ ] **Step 5: Look at the real output once**

```bash
cat > /tmp/peek.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func main() {
	fsys, _ := vfs.New(content.Files)
	p, _ := content.LoadProfile()
	s := shell.New(fsys, p, shell.Options{Width: 90, Interactive: true})
	for _, l := range os.Args[1:] {
		fmt.Println("$ " + l)
		fmt.Println(ansi.Strip(s.Run(l).Output))
	}
}
EOF
mkdir -p cmd/peek && mv /tmp/peek.go cmd/peek/main.go
go run ./cmd/peek "ls -l work" "cat work/engrave-me-now.md" "cat contact.md" neofetch help
rm -r cmd/peek
```

Expected: `4.9/5` stays on one line; bullet continuation lines are indented two spaces; each contact link appears once; `neofetch` shows the `KY` monogram beside the card. (Use `command cat` if `cat` is aliased.)

- [ ] **Step 6: Commit**

```bash
git add internal/shell
git commit -m "feat: add cat, open, neofetch, help and utility commands

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 6: Tab completion

**Files:**
- Create: `internal/shell/complete.go`
- Test: `internal/shell/complete_test.go`

**Interfaces:**
- Consumes: `visibleCommands()`, `Session.fs`, `Session.cwd` (Task 4).
- Produces: `func (s *Session) Complete(line string) (string, []string)` — `line` is the text before the cursor; returns the extended text and, when several names match and nothing more can be added, the candidates (directories suffixed `/`). A single match ends with `/` (directory) or a space.

- [ ] **Step 1: Write the failing test**

Create `internal/shell/complete_test.go`:

```go
package shell

import (
	"slices"
	"testing"
)

func TestComplete(t *testing.T) {
	s := newTestSession(t)
	cases := []struct {
		line, want string
		cands      []string
	}{
		{"ca", "cat ", nil},
		{"c", "c", []string{"cat", "cd", "clear"}},
		{"ne", "neofetch ", nil},
		{"zz", "zz", nil},
		{"cat ab", "cat about.md ", nil},
		{"cd w", "cd work/", nil},
		{"cat work/a", "cat work/acme.md ", nil},
		{"cat work/", "cat work/", []string{"acme.md", "beta.md"}},
		{"ls ~/pro", "ls ~/projects/", nil},
		{"ls /h", "ls /home/", nil},
		{"cat ", "cat ", []string{"projects/", "work/", "about.md", "contact.md"}},
		{"cat nope/x", "cat nope/x", nil},
		{"cat about.md/x", "cat about.md/x", nil},
		{"cat about.md c", "cat about.md contact.md ", nil},
		{"sud", "sud", nil}, // hidden commands are not offered
	}
	for _, c := range cases {
		got, cands := s.Complete(c.line)
		if got != c.want || !slices.Equal(cands, c.cands) {
			t.Errorf("Complete(%q) = %q %q, want %q %q", c.line, got, cands, c.want, c.cands)
		}
	}
}

func TestCompleteUsesCwd(t *testing.T) {
	s := newTestSession(t)
	s.Run("cd work")
	if got, _ := s.Complete("cat b"); got != "cat beta.md " {
		t.Fatalf("Complete in work = %q", got)
	}
	if got, _ := s.Complete("cat ../ab"); got != "cat ../about.md " {
		t.Fatalf("Complete ../ = %q", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/shell/ -run Complete`
Expected: FAIL — `s.Complete undefined`.

- [ ] **Step 3: Implement**

Create `internal/shell/complete.go`:

```go
package shell

import "strings"

// Complete extends the word being typed. line is the text before the cursor.
// It returns the new text and, when several names still match and nothing
// more can be added, the candidates to show (directories end in "/").
func (s *Session) Complete(line string) (string, []string) {
	i := strings.LastIndex(line, " ")
	word := line[i+1:]

	if strings.TrimSpace(line[:i+1]) == "" { // first word: a command name
		var names []string
		for _, c := range visibleCommands() {
			if strings.HasPrefix(c, word) {
				names = append(names, c)
			}
		}
		return extend(line, word, names, nil)
	}

	dir, base := "", word
	if j := strings.LastIndex(word, "/"); j >= 0 {
		dir, base = word[:j+1], word[j+1:]
	}
	n, err := s.fs.Resolve(s.cwd, dir)
	if err != nil || !n.Dir {
		return line, nil
	}
	var names []string
	dirs := map[string]bool{}
	for _, c := range n.Children() {
		if strings.HasPrefix(c.Name, base) {
			names = append(names, c.Name)
			dirs[c.Name] = c.Dir
		}
	}
	return extend(line, base, names, dirs)
}

// extend replaces the typed prefix with the longest completion. A single
// match is finished with "/" for a directory or a space otherwise.
func extend(line, typed string, names []string, dirs map[string]bool) (string, []string) {
	stem := line[:len(line)-len(typed)]
	switch len(names) {
	case 0:
		return line, nil
	case 1:
		if dirs[names[0]] {
			return stem + names[0] + "/", nil
		}
		return stem + names[0] + " ", nil
	}
	prefix := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	shown := make([]string, len(names))
	for i, n := range names {
		shown[i] = n
		if dirs[n] {
			shown[i] += "/"
		}
	}
	return stem + prefix, shown
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/shell/ -v -run Complete`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell
git commit -m "feat: add tab completion for commands and paths

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 7: Boot animation

**Files:**
- Create: `internal/boot/canvas.go`, `internal/boot/boot.go`, `internal/boot/testdata/*.golden` (generated)
- Test: `internal/boot/boot_test.go`

**Interfaces:**
- Consumes: `pixelfont.Join/Stack/Scale/HalfBlocks/MustText` (Task 1).
- Produces:
  - `const boot.Duration = 2100 * time.Millisecond`
  - `type boot.Options struct { Tagline string; Projects int }`
  - `func boot.Fits(w, h int) bool` — false below 40×12 or when no layout fits (2× needs ≥140 cols and ≥14 rows; 1× ≥72 cols and ≥11 rows; stacked ≥40 cols and ≥15 rows)
  - `func boot.Frame(t time.Duration, w, h int, o Options) string` — pure; exactly `h` lines of `w` columns; `""` for non-positive sizes

Timeline in `Frame`: pixel `i` of `n` lands at `400ms + 600ms·((i·37) mod n)/n`, shows `·` then `▪` (faint) then its block 120 ms later; before landing, its dot flies for 400 ms along a Bézier curve from a hashed screen-edge point with ease-out cubic. From 200 ms: tagline (muted), 40-column progress bar filling until 1800 ms, status line `mounting /home/guest` → `indexing N projects` (600 ms) → `starting shell` (1100 ms) → `[ ok ] ready` (1600 ms).

- [ ] **Step 1: Write the failing test**

Create `internal/boot/boot_test.go`:

```go
package boot

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

var opts = Options{Tagline: "CS student · data science intern at Cummins", Projects: 5}

func lines(frame string) []string {
	ls := strings.Split(ansi.Strip(frame), "\n")
	for i, l := range ls {
		ls[i] = strings.TrimRight(l, " ")
	}
	return ls
}

func TestFrameIsExactlyTheWindow(t *testing.T) {
	for _, size := range [][2]int{{160, 45}, {100, 30}, {60, 24}, {30, 10}} {
		for _, at := range []time.Duration{0, 500 * ms, Duration} {
			raw := strings.Split(Frame(at, size[0], size[1], opts), "\n")
			if len(raw) != size[1] {
				t.Fatalf("%v at %v: %d lines, want %d", size, at, len(raw), size[1])
			}
			for _, l := range raw {
				if w := ansi.StringWidth(l); w != size[0] {
					t.Fatalf("%v at %v: line width %d, want %d", size, at, w, size[0])
				}
			}
		}
	}
}

func contains(frame string, rows []string) bool {
	return strings.Contains(strings.Join(lines(frame), "\n"), strings.Join(rows, "\n"))
}

func TestFinalFramesShowTheName(t *testing.T) {
	oneX := []string{
		"█  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █    █   █ █   █ █▀▀▀▄ ▀▀█▀▀ █▀▀▀▀ █▀▀▀▄",
	}
	if f := Frame(Duration, 100, 30, opts); !contains(f, oneX) {
		t.Errorf("100×30 final frame lacks the 1× name:\n%s", strings.Join(lines(f), "\n"))
	}
	twoX := []string{"██      ██  ██      ██  ████████      ██████    ██      ██"}
	if f := Frame(Duration, 160, 45, opts); !contains(f, twoX) {
		t.Errorf("160×45 final frame lacks the 2× name")
	}
	stackedTop := []string{"   █  ▄▀ █   █ █▀▀▀▄ ▄▀▀▀▄ █   █"}
	if f := Frame(Duration, 60, 24, opts); !contains(f, stackedTop) {
		t.Errorf("60×24 final frame lacks the stacked name")
	}
}

func TestFirstFrameHasNoLetters(t *testing.T) {
	f := ansi.Strip(Frame(0, 100, 30, opts))
	if strings.ContainsAny(f, "█▀▄") {
		t.Fatal("letters visible at t=0")
	}
}

func TestStatusLine(t *testing.T) {
	cases := map[time.Duration]string{
		300 * ms:  "mounting /home/guest",
		800 * ms:  "indexing 5 projects",
		1300 * ms: "starting shell",
		1900 * ms: "[ ok ] ready",
	}
	for at, want := range cases {
		if f := ansi.Strip(Frame(at, 100, 30, opts)); !strings.Contains(f, want) {
			t.Errorf("at %v: missing %q", at, want)
		}
	}
	if f := ansi.Strip(Frame(100*ms, 100, 30, opts)); strings.Contains(f, "mounting") {
		t.Error("status shown before 200ms")
	}
}

func TestFits(t *testing.T) {
	cases := map[[2]int]bool{
		{30, 10}: false, {39, 30}: false, {60, 11}: false, {60, 14}: false,
		{60, 15}: true, {100, 12}: true, {100, 11}: false, {160, 45}: true,
	}
	for size, want := range cases {
		if got := Fits(size[0], size[1]); got != want {
			t.Errorf("Fits(%d, %d) = %v, want %v", size[0], size[1], got, want)
		}
	}
}

func TestDeterministic(t *testing.T) {
	a, b := Frame(700*ms, 100, 30, opts), Frame(700*ms, 100, 30, opts)
	if a != b {
		t.Fatal("same inputs gave different frames")
	}
}

// Review focus 3: zero and negative sizes never panic.
func TestDegenerateSizes(t *testing.T) {
	for _, size := range [][2]int{{0, 0}, {-1, 5}, {5, -1}, {1, 1}} {
		_ = Frame(time.Second, size[0], size[1], opts)
	}
	if Frame(time.Second, 0, 0, opts) != "" {
		t.Fatal("0×0 frame should be empty")
	}
}

// Review focus 1: the letters use the terminal's default color.
func TestLettersUseDefaultForeground(t *testing.T) {
	f := Frame(Duration, 100, 30, opts)
	if strings.Contains(f, "255;255;255") {
		t.Fatal("hard-coded white in frame")
	}
	for _, l := range strings.Split(f, "\n") {
		if i := strings.Index(l, "█"); i >= 0 && strings.LastIndex(l[:i], "\x1b[38") > strings.LastIndex(l[:i], "\x1b[m") {
			t.Fatalf("letter row is colored: %q", l)
		}
	}
}

// Golden frames make visual changes show up in review. After an intended
// change run: go test ./internal/boot -update, then look at testdata/.
func TestGoldenFrames(t *testing.T) {
	for _, size := range [][2]int{{160, 45}, {100, 30}, {60, 24}, {30, 10}} {
		for _, at := range []time.Duration{0, 500 * ms, 1000 * ms, 1500 * ms, Duration} {
			name := filepath.Join("testdata", fmt.Sprintf("%dx%d_%04dms.golden", size[0], size[1], at.Milliseconds()))
			got := strings.Join(lines(Frame(at, size[0], size[1], opts)), "\n") + "\n"
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("%v (run with -update to create)", err)
			}
			if got != string(want) {
				t.Errorf("%s changed; run with -update if intended", name)
			}
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/boot/`
Expected: FAIL — `undefined: Frame`, `undefined: Duration`, `undefined: Options`.

- [ ] **Step 3: Implement the canvas**

Create `internal/boot/canvas.go`:

```go
package boot

import "strings"

// tone is a cell's color. plain uses the terminal's default foreground.
type tone uint8

const (
	plain tone = iota
	faint      // #808080
	muted      // #a3a3a3
)

var sgr = map[tone]string{
	faint: "\x1b[38;2;128;128;128m",
	muted: "\x1b[38;2;163;163;163m",
}

// canvas is a w×h grid of styled cells.
type canvas struct {
	w, h  int
	cells [][]rune
	tones [][]tone
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, cells: make([][]rune, h), tones: make([][]tone, h)}
	for y := range h {
		c.cells[y] = []rune(strings.Repeat(" ", w))
		c.tones[y] = make([]tone, w)
	}
	return c
}

func (c *canvas) set(x, y int, r rune, t tone) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.cells[y][x], c.tones[y][x] = r, t
	}
}

func (c *canvas) empty(x, y int) bool {
	return x >= 0 && x < c.w && y >= 0 && y < c.h && c.cells[y][x] == ' '
}

// text writes s starting at x (each rune is one column).
func (c *canvas) text(x, y int, s string, t tone) {
	for i, r := range []rune(s) {
		c.set(x+i, y, r, t)
	}
}

// String renders rows joined by newlines, emitting a color code only where
// the tone changes so frames stay small.
func (c *canvas) String() string {
	var b strings.Builder
	for y := range c.h {
		if y > 0 {
			b.WriteByte('\n')
		}
		cur := plain
		for x := range c.w {
			if t := c.tones[y][x]; t != cur {
				if cur != plain {
					b.WriteString("\x1b[m")
				}
				if t != plain {
					b.WriteString(sgr[t])
				}
				cur = t
			}
			b.WriteRune(c.cells[y][x])
		}
		if cur != plain {
			b.WriteString("\x1b[m")
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Implement the animation**

Create `internal/boot/boot.go`:

```go
// Package boot draws the startup animation: faint dots fly in from the
// screen edges and settle into "KUDAY YURTER" in the site's pixel font, while
// a tagline, progress bar and status line fill in underneath.
//
// Frame is a pure function of time and window size, so the animation can be
// tested frame by frame and survives resizes without restarting.
package boot

import (
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/namelessmonarch0/ssh-portfolio/internal/pixelfont"
)

// Duration is how long the animation runs before the shell takes over.
const Duration = 2100 * time.Millisecond

const (
	ms     = time.Millisecond
	flight = 400 * ms // how long each dot takes to reach its pixel
	below  = 5        // rows under the name: blank, tagline, blank, bar, status
)

// Options are the words drawn under the name.
type Options struct {
	Tagline  string
	Projects int // shown as "indexing N projects"
}

var (
	fullName = pixelfont.Join(4, pixelfont.MustText("KUDAY"), pixelfont.MustText("YURTER"))
	stacked  = pixelfont.Stack(1, pixelfont.MustText("KUDAY"), pixelfont.MustText("YURTER"))

	// layouts are tried in order; the first that fits the window wins.
	layouts = []struct {
		minW int
		art  []string
	}{
		{140, pixelfont.HalfBlocks(pixelfont.Scale(fullName, 2))}, // 136×7
		{72, pixelfont.HalfBlocks(fullName)},                      // 68×4
		{40, pixelfont.HalfBlocks(stacked)},                       // 35×8
	}
)

func pickLayout(w, h int) ([]string, bool) {
	if w < 40 || h < 12 {
		return nil, false
	}
	for _, l := range layouts {
		if w >= l.minW && h >= len(l.art)+below+2 {
			return l.art, true
		}
	}
	return nil, false
}

// Fits reports whether a w×h window is big enough for the animation.
func Fits(w, h int) bool {
	_, ok := pickLayout(w, h)
	return ok
}

type target struct {
	x, y int
	r    rune
}

// Frame draws the animation at time t for a w×h window: exactly h lines of
// w columns. Windows too small for any layout get a blank frame.
func Frame(t time.Duration, w, h int, o Options) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	c := newCanvas(w, h)
	art, ok := pickLayout(w, h)
	if !ok {
		return c.String()
	}
	artW, artH := utf8.RuneCountInString(art[0]), len(art)
	top, left := (h-(artH+below))/2, (w-artW)/2

	var targets []target
	for y, row := range art {
		for x, r := range []rune(row) {
			if r != ' ' {
				targets = append(targets, target{left + x, top + y, r})
			}
		}
	}

	n := len(targets)
	for i, tg := range targets {
		// Same scatter as the site's pixel reveal: index × 37 mod total.
		k := (i * 37) % n
		land := 400*ms + time.Duration(float64(600*ms)*float64(k)/float64(n))
		switch {
		case t >= land+120*ms:
			c.set(tg.x, tg.y, tg.r, plain)
		case t >= land+60*ms:
			c.set(tg.x, tg.y, '▪', faint)
		case t >= land:
			c.set(tg.x, tg.y, '·', faint)
		case t >= land-flight:
			u := float64(t-(land-flight)) / float64(flight)
			x, y := dot(i, tg, w, h, u)
			if c.empty(x, y) {
				c.set(x, y, '·', faint)
			}
		}
	}

	if t >= 200*ms {
		y := top + artH + 1
		c.text((w-utf8.RuneCountInString(o.Tagline))/2, y, o.Tagline, muted)

		barW := min(40, w-4)
		p := math.Min(1, float64(t-200*ms)/float64(1600*ms))
		filled := int(p*float64(barW) + 0.5)
		bx := (w - barW) / 2
		for i := range barW {
			if i < filled {
				c.set(bx+i, y+2, '━', plain)
			} else {
				c.set(bx+i, y+2, '─', faint)
			}
		}

		st := status(t, o.Projects)
		c.text((w-utf8.RuneCountInString(st))/2, y+3, st, faint)
	}
	return c.String()
}

func status(t time.Duration, projects int) string {
	switch {
	case t < 600*ms:
		return "mounting /home/guest"
	case t < 1100*ms:
		return fmt.Sprintf("indexing %d projects", projects)
	case t < 1600*ms:
		return "starting shell"
	}
	return "[ ok ] ready"
}

// dot is where dot i is at progress u∈[0,1): it starts on a screen edge
// chosen by a hash of i and follows a curved path (quadratic Bézier) to its
// pixel, slowing as it arrives (ease-out cubic).
func dot(i int, tg target, w, h int, u float64) (int, int) {
	hash := uint32(i)*2654435761 + 12345
	pos := float64((hash>>8)%1000) / 1000
	var sx, sy float64
	switch hash % 4 {
	case 0:
		sx, sy = pos*float64(w-1), 0
	case 1:
		sx, sy = float64(w-1), pos*float64(h-1)
	case 2:
		sx, sy = pos*float64(w-1), float64(h-1)
	default:
		sx, sy = 0, pos*float64(h-1)
	}
	tx, ty := float64(tg.x), float64(tg.y)
	bend := 0.35
	if hash&(1<<20) != 0 {
		bend = -bend
	}
	cx, cy := (sx+tx)/2-(ty-sy)*bend, (sy+ty)/2+(tx-sx)*bend

	e := 1 - math.Pow(1-u, 3)
	x := (1-e)*(1-e)*sx + 2*(1-e)*e*cx + e*e*tx
	y := (1-e)*(1-e)*sy + 2*(1-e)*e*cy + e*e*ty
	return int(math.Round(x)), int(math.Round(y))
}
```

- [ ] **Step 5: Generate the golden frames and look at them**

```bash
go test ./internal/boot/ -update
command cat internal/boot/testdata/100x30_0500ms.golden internal/boot/testdata/100x30_2100ms.golden
```

Expected: the 500 ms frame shows scattered `·` dots converging on a half-lit name with `mounting /home/guest` under a partly filled bar; the 2100 ms frame shows the full `KUDAY YURTER` name, the tagline, a full bar, and `[ ok ] ready`, all centered. The `30x10` files are blank (too small).

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/boot/ -v`
Expected: PASS (8 tests).

- [ ] **Step 7: Commit**

```bash
git add internal/boot
git commit -m "feat: add pixel-name boot animation

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 8: Terminal UI (boot ↔ inline shell)

**Files:**
- Create: `internal/ui/ui.go`
- Test: `internal/ui/ui_test.go`

**Interfaces:**
- Consumes: `shell.Session` (`Run`, `Prompt`, `History`, `SetWidth`, `Complete`), `shell.Action*` (Tasks 4–6); `boot.Frame`, `boot.Fits`, `boot.Duration`, `boot.Options` (Task 7); `style.Faint`, `style.Muted` (Task 3).
- Produces:
  - `type ui.Options struct { Width, Height int; Plain bool; Boot boot.Options; Welcome string; OnCommand func(string); Print func(string) tea.Cmd }` — `Print` nil means `tea.Println` (tests inject a recorder)
  - `func ui.New(sh *shell.Session, o Options) Model`; `Model` implements `tea.Model`
  - `type ui.ShutdownMsg struct{}` — prints `system going down for update, reconnect in a moment` and quits

Behavior: boot runs in the alt screen and ticks at 30 fps; any key or paste during boot only skips it. The shell view is inline: a faint hint `try: ls · cat about.md · help` until the first command, then the prompt with the real cursor, hard-wrapped to the window. Each command prints `prompt + line` and its output above the view.

**Why output is printed in pieces:** Bubble Tea inserts printed text above the view in one step. If that text is taller than the window, the insert pushes the prompt off the bottom and the renderer loses track of it (seen in manual testing: after `cat work/cummins.md` in a 30-row window the prompt text vanished). `Model.print` hard-wraps to the window width and sends pieces of at most `height − 3` rows; `TestTallOutputIsPrintedInPiecesThatFit` pins this.

- [ ] **Step 1: Add the dependency**

```bash
go get charm.land/bubbletea/v2@v2.0.10
```

- [ ] **Step 2: Write the failing tests**

Create `internal/ui/ui_test.go`:

```go
package ui

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

type harness struct {
	t       *testing.T
	m       Model
	printed []string
	quit    bool
}

func newHarness(t *testing.T, o Options) *harness {
	t.Helper()
	fsys, err := vfs.New(fstest.MapFS{
		"about.md":     {Data: []byte("---\ntitle: About\nsummary: Who\n---\nHi.\n")},
		"work/acme.md": {Data: []byte("---\ntitle: Acme\nsummary: Job\n---\nWork.\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t}
	o.Welcome = "welcome!"
	o.Print = func(s string) tea.Cmd {
		h.printed = append(h.printed, ansi.Strip(s))
		return nil
	}
	sh := shell.New(fsys, content.Profile{}, shell.Options{Width: o.Width, Interactive: true})
	h.m = New(sh, o)
	h.m.Init()
	return h
}

// send runs one message through Update and executes the returned command
// to notice tea.Quit. The recording Print returns nil, so tea.Sequence
// collapses to the single remaining command.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	next, cmd := h.m.Update(msg)
	h.m = next.(Model)
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			h.quit = true
		}
	}
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		h.send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func (h *harness) press(code rune, mod ...tea.KeyMod) {
	k := tea.KeyPressMsg{Code: code}
	for _, m := range mod {
		k.Mod |= m
	}
	h.send(k)
}

func (h *harness) last() string {
	if len(h.printed) == 0 {
		return ""
	}
	return h.printed[len(h.printed)-1]
}

func TestSmallWindowSkipsBoot(t *testing.T) {
	h := newHarness(t, Options{Width: 30, Height: 10})
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("30×10 should start in the shell with the welcome: mode %v printed %q", h.m.mode, h.printed)
	}
}

func TestPlainSkipsBootAndWelcomes(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30, Plain: true})
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("plain terminal should start in the shell: mode %v printed %q", h.m.mode, h.printed)
	}
}

func TestBootEndsAfterDuration(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	if h.m.mode != modeBoot {
		t.Fatal("should start in boot mode")
	}
	t0 := time.Unix(0, 0)
	h.send(tickMsg(t0))
	h.send(tickMsg(t0.Add(time.Second)))
	if h.m.mode != modeBoot {
		t.Fatal("left boot too early")
	}
	if !h.m.View().AltScreen {
		t.Fatal("boot should use the alt screen")
	}
	h.send(tickMsg(t0.Add(boot.Duration)))
	if h.m.mode != modeShell || h.last() != "welcome!" {
		t.Fatalf("mode %v, printed %q", h.m.mode, h.printed)
	}
	if h.m.View().AltScreen {
		t.Fatal("shell should be inline")
	}
}

// Review focus 5: input during boot skips and is swallowed.
func TestKeysAndPastesDuringBootAreSwallowed(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.typeText("x")
	if h.m.mode != modeShell || len(h.m.input) != 0 {
		t.Fatalf("key during boot: mode %v input %q", h.m.mode, string(h.m.input))
	}
	h = newHarness(t, Options{Width: 100, Height: 30})
	h.send(tea.PasteMsg{Content: "rm -rf /"})
	if h.m.mode != modeShell || len(h.m.input) != 0 {
		t.Fatalf("paste during boot: mode %v input %q", h.m.mode, string(h.m.input))
	}
}

func TestRunCommand(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	var logged []string
	h.m.o.OnCommand = func(l string) { logged = append(logged, l) }
	if !strings.Contains(ansi.Strip(h.m.View().Content), "try: ls") {
		t.Fatal("hint missing before the first command")
	}
	h.typeText("ls")
	h.press(tea.KeyEnter)
	if got := h.last(); got != "guest@kuday:~$ ls\nwork/  about.md" {
		t.Fatalf("printed %q", got)
	}
	if len(h.m.input) != 0 || strings.Contains(ansi.Strip(h.m.View().Content), "try: ls") {
		t.Fatal("input not reset or hint still shown")
	}
	if len(logged) != 1 || logged[0] != "ls" {
		t.Fatalf("logged %q", logged)
	}
	h.typeText("cd work")
	h.press(tea.KeyEnter)
	if got := h.last(); got != "guest@kuday:~$ cd work" {
		t.Fatalf("echo should use the old cwd: %q", got)
	}
	if got := ansi.Strip(h.m.View().Content); got != "guest@kuday:~/work$ " {
		t.Fatalf("view = %q", got)
	}
}

func TestLineEditing(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.typeText("cat abut")
	h.press(tea.KeyLeft)
	h.press(tea.KeyLeft)
	h.typeText("o")
	if string(h.m.input) != "cat about" {
		t.Fatalf("insert at cursor: %q", string(h.m.input))
	}
	h.press(tea.KeyBackspace)
	h.press(tea.KeyHome)
	h.press(tea.KeyDelete)
	if string(h.m.input) != "at abut" {
		t.Fatalf("backspace/delete: %q", string(h.m.input))
	}
	h.press(tea.KeyEnd)
	h.press('w', tea.ModCtrl)
	if string(h.m.input) != "at " {
		t.Fatalf("ctrl+w: %q", string(h.m.input))
	}
	h.press('u', tea.ModCtrl)
	if string(h.m.input) != "" {
		t.Fatalf("ctrl+u: %q", string(h.m.input))
	}
	h.typeText("pwd")
	h.press('c', tea.ModCtrl)
	if len(h.m.input) != 0 || h.last() != "guest@kuday:~$ pwd^C" {
		t.Fatalf("ctrl+c: input %q printed %q", string(h.m.input), h.last())
	}
}

func TestHistory(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	for _, l := range []string{"pwd", "ls"} {
		h.typeText(l)
		h.press(tea.KeyEnter)
	}
	h.typeText("dra")
	h.press(tea.KeyUp)
	if string(h.m.input) != "ls" {
		t.Fatalf("up: %q", string(h.m.input))
	}
	h.press(tea.KeyUp)
	h.press(tea.KeyUp) // stays at the oldest
	if string(h.m.input) != "pwd" {
		t.Fatalf("up up: %q", string(h.m.input))
	}
	h.press(tea.KeyDown)
	h.press(tea.KeyDown)
	if string(h.m.input) != "dra" {
		t.Fatalf("down restores draft: %q", string(h.m.input))
	}
}

func TestTabCompletion(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.typeText("ca")
	h.press(tea.KeyTab)
	if string(h.m.input) != "cat " {
		t.Fatalf("tab: %q", string(h.m.input))
	}
	h.press(tea.KeyTab)
	n := len(h.printed)
	h.press(tea.KeyTab)
	if len(h.printed) != n+1 || !strings.HasSuffix(h.last(), "work/  about.md") {
		t.Fatalf("second tab should list candidates: %q", h.printed)
	}
}

func TestClearAndExit(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.press('d', tea.ModCtrl)
	if !h.quit || !strings.HasSuffix(h.last(), "exit\nlogout") {
		t.Fatalf("ctrl+d: quit %v printed %q", h.quit, h.last())
	}
}

func TestBootCommandReplays(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.typeText("x") // skip first boot
	h.typeText("boot")
	h.press(tea.KeyEnter)
	if h.m.mode != modeBoot {
		t.Fatal("boot should replay the animation")
	}
	h.typeText("q")
	if h.m.mode != modeShell || h.last() != "guest@kuday:~$ boot" {
		t.Fatalf("second skip: mode %v printed %q", h.m.mode, h.printed)
	}

	small := newHarness(t, Options{Width: 30, Height: 10})
	small.typeText("boot")
	small.press(tea.KeyEnter)
	if small.m.mode != modeShell || !strings.Contains(strings.ReplaceAll(small.last(), "\n", ""), "can't show the animation") {
		t.Fatalf("small boot: %q", small.last())
	}
}

func TestShutdownPrintsAndQuits(t *testing.T) {
	h := newHarness(t, Options{Width: 100, Height: 30})
	h.send(ShutdownMsg{})
	if !h.quit || !strings.Contains(h.last(), "system going down") {
		t.Fatalf("shutdown: quit %v printed %q", h.quit, h.printed)
	}
}

// Review focus 2: pasted newlines and escapes never reach the command line.
func TestPasteIsCleaned(t *testing.T) {
	h := newHarness(t, Options{Width: 80, Height: 24, Plain: true})
	h.send(tea.PasteMsg{Content: "echo a\nb\x1b[31m\tc"})
	if got := string(h.m.input); got != "echo a b[31m c" {
		t.Fatalf("pasted = %q", got)
	}
	h.send(tea.PasteMsg{Content: strings.Repeat("x", 5000)})
	if len(h.m.input) != maxInput {
		t.Fatalf("input length %d, want cap %d", len(h.m.input), maxInput)
	}
}

// Review focus 3: tiny or unknown widths wrap instead of panicking.
func TestViewWrapsInNarrowWindows(t *testing.T) {
	for _, w := range []int{0, 1, 10} {
		h := newHarness(t, Options{Width: w, Height: 24, Plain: true})
		h.send(tea.WindowSizeMsg{Width: w, Height: 24})
		h.typeText(strings.Repeat("a", 300))
		v := h.m.View()
		limit := w
		if w <= 0 {
			limit = 80
		}
		for _, l := range strings.Split(ansi.Strip(v.Content), "\n") {
			if ansi.StringWidth(l) > limit {
				t.Fatalf("width %d: line of %d columns", w, ansi.StringWidth(l))
			}
		}
		if v.Cursor == nil || v.Cursor.X >= limit {
			t.Fatalf("width %d: bad cursor %+v", w, v.Cursor)
		}
	}
}

func TestTallOutputIsPrintedInPiecesThatFit(t *testing.T) {
	h := newHarness(t, Options{Width: 40, Height: 10, Plain: true})
	h.typeText("echo " + strings.Repeat("word ", 60)) // one 300-column line: 8 rows at width 40
	h.press(tea.KeyEnter)
	h.typeText("history")
	h.press(tea.KeyEnter)
	for _, p := range h.printed {
		rows := 0
		for _, l := range strings.Split(p, "\n") {
			rows += 1 + max(0, ansi.StringWidth(l)-1)/40
		}
		if rows > 7 {
			t.Fatalf("printed piece of %d rows in a 10-row window:\n%s", rows, p)
		}
	}
	all := strings.ReplaceAll(strings.Join(h.printed, ""), "\n", "") // undo wrapping
	if !strings.Contains(all, "1  echo word") || !strings.Contains(all, "2  history") || strings.Count(all, "word") != 180 {
		t.Fatalf("output lost when split:\n%s", all)
	}
}
```

`TestKeysAndPastesDuringBootAreSwallowed` pins Review Focus 5, `TestPasteIsCleaned` Review Focus 2, and `TestViewWrapsInNarrowWindows` Review Focus 3.

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./internal/ui/`
Expected: FAIL — `undefined: New`, `undefined: Model`, `undefined: Options`.

- [ ] **Step 4: Implement**

Create `internal/ui/ui.go`:

```go
// Package ui is the Bubble Tea program each visitor runs: the boot animation
// in the alternate screen, then an inline shell whose output is printed into
// the terminal's own scrollback (so scrolling and copying work natively).
package ui

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
)

// ShutdownMsg tells a session the server is going down.
type ShutdownMsg struct{}

type tickMsg time.Time

type mode int

const (
	modeBoot mode = iota
	modeShell
)

const maxInput = 1024 // runes; longer pastes are cut

// Options configure a Model.
type Options struct {
	Width, Height int
	Plain         bool                 // no animation (colorless or dumb terminal)
	Boot          boot.Options         // words under the name
	Welcome       string               // printed when the shell starts
	OnCommand     func(line string)    // called for every non-blank line (logging)
	Print         func(string) tea.Cmd // nil means tea.Println; tests record output
}

// Model is one visitor's screen.
type Model struct {
	sh       *shell.Session
	o        Options
	mode     mode
	w, h     int
	start    time.Time // first tick of the current boot run
	now      time.Time // latest tick
	welcomed bool
	input    []rune
	cursor   int
	hist     int    // history index while browsing; len(history) when on a fresh line
	draft    []rune // the fresh line saved while browsing history
	tabbed   bool   // previous key was a Tab that had several candidates
	used     bool   // a command has run, so the hint is hidden
}

// New decides whether to animate: plain terminals and windows too small for
// the animation start straight in the shell.
func New(sh *shell.Session, o Options) Model {
	if o.Print == nil {
		o.Print = func(s string) tea.Cmd { return tea.Println(s) }
	}
	m := Model{sh: sh, o: o, w: o.Width, h: o.Height}
	if o.Plain || !boot.Fits(o.Width, o.Height) {
		m.mode = modeShell
		m.welcomed = true // Init prints it
	}
	return m
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init starts the animation, or prints the welcome when there is none.
func (m Model) Init() tea.Cmd {
	if m.mode == modeBoot {
		return tick()
	}
	return m.print(m.o.Welcome)
}

// Update handles one message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.sh.SetWidth(msg.Width)
		return m, nil

	case ShutdownMsg:
		m.mode = modeShell
		return m, tea.Sequence(m.print(style.Muted.Render("system going down for update, reconnect in a moment")), tea.Quit)

	case tickMsg:
		if m.mode != modeBoot {
			return m, nil
		}
		now := time.Time(msg)
		if m.start.IsZero() {
			m.start = now
		}
		m.now = now
		if now.Sub(m.start) >= boot.Duration {
			return m.enterShell()
		}
		return m, tick()

	case tea.PasteMsg:
		if m.mode == modeBoot {
			return m.enterShell() // swallow: a paste only skips the animation
		}
		m.insert(clean(msg.Content))
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeBoot {
			return m.enterShell() // swallow: any key only skips the animation
		}
		return m.key(msg)
	}
	return m, nil
}

func (m Model) enterShell() (tea.Model, tea.Cmd) {
	m.mode = modeShell
	m.start, m.now = time.Time{}, time.Time{}
	if m.welcomed {
		return m, nil
	}
	m.welcomed = true
	return m, m.print(m.o.Welcome)
}

func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	wasTab := m.tabbed
	m.tabbed = false
	switch k.String() {
	case "enter":
		return m.submit()
	case "ctrl+c":
		echo := m.promptLine() + "^C"
		m.resetLine()
		return m, m.print(echo)
	case "ctrl+d":
		if len(m.input) == 0 {
			m.input = []rune("exit")
			return m.submit()
		}
	case "ctrl+l":
		return m, tea.ClearScreen
	case "tab":
		return m.complete(wasTab)
	case "backspace", "ctrl+h":
		if m.cursor > 0 {
			m.input = slices.Delete(m.input, m.cursor-1, m.cursor)
			m.cursor--
		}
	case "delete":
		if m.cursor < len(m.input) {
			m.input = slices.Delete(m.input, m.cursor, m.cursor+1)
		}
	case "left", "ctrl+b":
		m.cursor = max(0, m.cursor-1)
	case "right", "ctrl+f":
		m.cursor = min(len(m.input), m.cursor+1)
	case "home", "ctrl+a":
		m.cursor = 0
	case "end", "ctrl+e":
		m.cursor = len(m.input)
	case "ctrl+u":
		m.input = slices.Clone(m.input[m.cursor:])
		m.cursor = 0
	case "ctrl+w":
		i := m.cursor
		for i > 0 && m.input[i-1] == ' ' {
			i--
		}
		for i > 0 && m.input[i-1] != ' ' {
			i--
		}
		m.input = slices.Delete(m.input, i, m.cursor)
		m.cursor = i
	case "up":
		m.historyPrev()
	case "down":
		m.historyNext()
	default:
		if t := k.Key().Text; t != "" {
			m.insert(clean(t))
		}
	}
	return m, nil
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	line := string(m.input)
	echo := m.promptLine() // uses the directory the command was typed in
	m.resetLine()
	if strings.TrimSpace(line) != "" {
		m.used = true
		if m.o.OnCommand != nil {
			m.o.OnCommand(line)
		}
	}
	res := m.sh.Run(line)
	m.hist = len(m.sh.History())

	switch res.Action {
	case shell.ActionClear:
		return m, tea.ClearScreen
	case shell.ActionExit:
		return m, tea.Sequence(m.print(echo+"\n"+res.Output), tea.Quit)
	case shell.ActionBoot:
		if m.o.Plain || !boot.Fits(m.w, m.h) {
			return m, m.print(echo + "\n" + "boot: this window can't show the animation (too small or no color)")
		}
		m.mode = modeBoot
		return m, tea.Sequence(m.print(echo), tick())
	}
	if res.Output != "" {
		echo += "\n" + res.Output
	}
	return m, m.print(echo)
}

func (m Model) complete(wasTab bool) (tea.Model, tea.Cmd) {
	before, after := string(m.input[:m.cursor]), m.input[m.cursor:]
	got, cands := m.sh.Complete(before)
	if got != before {
		m.input = append([]rune(got), after...)
		m.cursor = len([]rune(got))
		return m, nil
	}
	if len(cands) < 2 {
		return m, nil
	}
	m.tabbed = true
	if !wasTab {
		return m, nil // like bash: the second Tab lists the candidates
	}
	return m, m.print(m.promptLine() + "\n" + strings.Join(cands, "  "))
}

// print shows s above the prompt. Bubble Tea inserts printed text above the
// view in one go, and a piece taller than the window pushes the prompt off
// the screen, after which the renderer loses track of it. So output is
// hard-wrapped to the window and sent in pieces that each fit.
func (m Model) print(s string) tea.Cmd {
	room := m.h - 3 // leave the hint and prompt lines on screen
	if m.h <= 0 {
		room = 20
	}
	room = max(1, room)
	lines := strings.Split(ansi.Hardwrap(s, m.width(), true), "\n")
	var cmds []tea.Cmd
	for len(lines) > 0 {
		n := min(room, len(lines))
		cmds = append(cmds, m.o.Print(strings.Join(lines[:n], "\n")))
		lines = lines[n:]
	}
	return tea.Sequence(cmds...)
}

func (m *Model) insert(s string) {
	r := []rune(s)
	if room := maxInput - len(m.input); len(r) > room {
		r = r[:max(0, room)]
	}
	m.input = slices.Insert(m.input, m.cursor, r...)
	m.cursor += len(r)
}

func (m *Model) resetLine() {
	m.input, m.cursor, m.draft = nil, 0, nil
	m.hist = len(m.sh.History())
}

func (m *Model) setInput(r []rune) {
	m.input = slices.Clone(r)
	m.cursor = len(m.input)
}

func (m *Model) historyPrev() {
	h := m.sh.History()
	if m.hist > len(h) {
		m.hist = len(h)
	}
	if m.hist == 0 {
		return
	}
	if m.hist == len(h) {
		m.draft = slices.Clone(m.input)
	}
	m.hist--
	m.setInput([]rune(h[m.hist]))
}

func (m *Model) historyNext() {
	h := m.sh.History()
	if m.hist >= len(h) {
		return
	}
	m.hist++
	if m.hist == len(h) {
		m.setInput(m.draft)
	} else {
		m.setInput([]rune(h[m.hist]))
	}
}

// clean makes pasted or typed text safe for a single command line: line
// breaks and tabs become spaces, other control characters are dropped.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, s)
}

func (m Model) promptLine() string { return m.sh.Prompt() + string(m.input) }

func (m Model) width() int {
	if m.w <= 0 {
		return 80
	}
	return m.w
}

// View is the animation in boot mode, otherwise the hint (until the first
// command) and the prompt with the cursor, hard-wrapped to the window.
func (m Model) View() tea.View {
	if m.mode == modeBoot {
		v := tea.NewView(boot.Frame(m.now.Sub(m.start), m.w, m.h, m.o.Boot))
		v.AltScreen = true
		return v
	}
	w := m.width()
	var b strings.Builder
	top := 0
	if !m.used {
		b.WriteString(style.Faint.Render(ansi.Truncate("try: ls · cat about.md · help", w, "")) + "\n")
		top = 1
	}
	b.WriteString(ansi.Hardwrap(m.promptLine(), w, true))
	v := tea.NewView(b.String())
	pos := ansi.StringWidth(m.sh.Prompt()) + ansi.StringWidth(string(m.input[:m.cursor]))
	v.Cursor = tea.NewCursor(pos%w, top+pos/w)
	return v
}
```

- [ ] **Step 5: Run the tests**

Run: `go vet ./internal/ui/ && go test ./internal/ui/ -v`
Expected: PASS (15 tests).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/ui
git commit -m "feat: add terminal UI with boot and inline shell

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 9: SSH server, limits, and entry point

**Files:**
- Create: `internal/server/config.go`, `internal/server/limits.go`, `internal/server/registry.go`, `internal/server/server.go`, `cmd/server/main.go`
- Test: `internal/server/server_test.go`

**Interfaces:**
- Consumes: `vfs.New`, `vfs.Home`, `content.Files`, `content.LoadProfile` (Task 2); `shell.New`, `shell.Options`, `shell.Result` (Task 4); `boot.Options` (Task 7); `ui.New`, `ui.Options`, `ui.ShutdownMsg` (Task 8); `style.Bold/Muted` (Task 3).
- Produces:
  - `type server.Config struct { ListenAddr, HostKeyPath, PublicHost string; MaxSessions, PerIP int; IdleTimeout, MaxSession time.Duration }`; `func server.ConfigFromEnv(getenv func(string) string) (Config, error)`
  - `func server.New(cfg Config, fsys *vfs.FS, profile content.Profile, log *slog.Logger) (*Server, error)`; `Server.SSH *ssh.Server`; `func (s *Server) Shutdown(ctx context.Context) error`

Middleware order matters: `wish.WithMiddleware` wraps in list order, so the **last** entry runs **first**: recover → log → rate limit → session limits → registry cleanup → Bubble Tea (PTY sessions) → exec mode (no PTY). The Bubble Tea middleware calls the next handler after the program exits, so exec mode must skip sessions that have a PTY. No auth handlers are set, which makes the SSH server accept the `none` method: visitors need no password or key.

- [ ] **Step 1: Add dependencies**

```bash
go get charm.land/wish/v2@v2.0.4 charm.land/ssh@v0.4.3 github.com/charmbracelet/colorprofile@v0.4.3 golang.org/x/time/rate golang.org/x/crypto/ssh
```

- [ ] **Step 2: Write the failing integration tests**

These start a real server on a random port and connect with `golang.org/x/crypto/ssh` using no credentials.

Create `internal/server/server_test.go`:

```go
package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/wish/v2/testsession"
	"github.com/charmbracelet/x/ansi"
	gossh "golang.org/x/crypto/ssh"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func testConfig(t *testing.T) Config {
	return Config{
		ListenAddr:  "127.0.0.1:0",
		HostKeyPath: filepath.Join(t.TempDir(), "keys", "host_ed25519"),
		PublicHost:  "term.test",
		MaxSessions: 100,
		PerIP:       5,
		IdleTimeout: time.Minute,
		MaxSession:  time.Minute,
	}
}

func newServer(t *testing.T, cfg Config) *Server {
	t.Helper()
	fsys, err := vfs.New(content.Files)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := content.LoadProfile()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(cfg, fsys, profile, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func startServer(t *testing.T, cfg Config) (*Server, string) {
	t.Helper()
	srv := newServer(t, cfg)
	return srv, testsession.Listen(t, srv.SSH)
}

// noAuth is a client with no credentials at all, like a visitor without keys.
func noAuth() *gossh.ClientConfig {
	return &gossh.ClientConfig{User: "anyone", HostKeyCallback: gossh.InsecureIgnoreHostKey()}
}

func session(t *testing.T, addr string) *gossh.Session {
	t.Helper()
	s, err := testsession.NewClientSession(t, addr, noAuth())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func exitStatus(err error) int {
	var ee *gossh.ExitError
	if errors.As(err, &ee) {
		return ee.ExitStatus()
	}
	if err != nil {
		return -1
	}
	return 0
}

func TestExecModePrintsPlainText(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	out, err := session(t, addr).Output("cat about.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Texas A&M University–Victoria") {
		t.Fatalf("output:\n%s", out)
	}
	if bytes.Contains(out, []byte("\x1b")) {
		t.Fatalf("exec output has escape sequences: %q", out)
	}
}

func TestExecModeExitCodes(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	if _, err := session(t, addr).Output("nope"); exitStatus(err) != 127 {
		t.Fatalf("unknown command exit = %d (%v)", exitStatus(err), err)
	}
	if _, err := session(t, addr).Output("cat work"); exitStatus(err) != 1 {
		t.Fatalf("cat dir exit = %d (%v)", exitStatus(err), err)
	}
}

func TestShellWithoutTerminalPrintsAboutAndHint(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	s := session(t, addr)
	var out bytes.Buffer
	s.Stdout = &out
	if err := s.Shell(); err != nil {
		t.Fatal(err)
	}
	if err := s.Wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "About") || !strings.Contains(out.String(), "ssh -t term.test") {
		t.Fatalf("output:\n%s", out.String())
	}
}

// syncBuffer collects terminal output from the SSH client goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitFor(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(ansi.Strip(out.String()), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q; got:\n%s", want, ansi.Strip(out.String()))
}

func interactive(t *testing.T, addr string, w, h int) (*gossh.Session, io.Writer, *syncBuffer) {
	t.Helper()
	s := session(t, addr)
	if err := s.RequestPty("xterm-256color", h, w, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	stdin, err := s.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out := &syncBuffer{}
	s.Stdout = out
	if err := s.Shell(); err != nil {
		t.Fatal(err)
	}
	return s, stdin, out
}

func TestInteractiveSession(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	s, stdin, out := interactive(t, addr, 100, 30)

	waitFor(t, out, "mounting") // the boot animation is running
	io.WriteString(stdin, "x")  // any key skips it
	waitFor(t, out, "guest@kuday")
	io.WriteString(stdin, "cat contact.md\r")
	waitFor(t, out, "kudayyurter@gmail.com")
	io.WriteString(stdin, "exit\r")

	done := make(chan error, 1)
	go func() { done <- s.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("session ended with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session did not end after exit")
	}
}

func TestSmallTerminalSkipsBoot(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	_, _, out := interactive(t, addr, 30, 10)
	waitFor(t, out, "guest@kuday")
	if strings.Contains(ansi.Strip(out.String()), "mounting") {
		t.Fatal("boot animation shown in a 30×10 terminal")
	}
}

func TestSessionLimit(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxSessions = 1
	_, addr := startServer(t, cfg)
	_, _, out := interactive(t, addr, 100, 30)
	waitFor(t, out, "mounting")

	got, err := session(t, addr).Output("pwd")
	if exitStatus(err) != 1 || !strings.Contains(string(got), "server busy") {
		t.Fatalf("second session: %q exit %d", got, exitStatus(err))
	}
}

func TestShutdownWarnsOpenSessions(t *testing.T) {
	srv, addr := startServer(t, testConfig(t))
	sess, stdin, out := interactive(t, addr, 100, 30)
	waitFor(t, out, "mounting")
	io.WriteString(stdin, "x")
	waitFor(t, out, "guest@kuday")
	for srv.programs.count() == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Shutdown waits for clients to hang up; a real ssh client does that
	// when its session ends, so only check that the session ends.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go srv.Shutdown(ctx)
	waitFor(t, out, "system going down")

	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("session still open after shutdown")
	}
}

func TestHostKeyIsCreatedOnceAndReused(t *testing.T) {
	cfg := testConfig(t)
	newServer(t, cfg)
	first, err := os.ReadFile(cfg.HostKeyPath)
	if err != nil {
		t.Fatalf("host key not written: %v", err)
	}
	newServer(t, cfg)
	second, _ := os.ReadFile(cfg.HostKeyPath)
	if !bytes.Equal(first, second) {
		t.Fatal("host key changed between starts")
	}
}

func TestConfigFromEnv(t *testing.T) {
	c, err := ConfigFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr != ":2222" || c.HostKeyPath != "/data/ssh_host_ed25519" || c.MaxSessions != 100 ||
		c.IdleTimeout != 10*time.Minute || c.MaxSession != 30*time.Minute || c.PublicHost != "term.kudayyurter.dev" || c.PerIP != 5 {
		t.Fatalf("defaults = %+v", c)
	}
	env := map[string]string{"MAX_SESSIONS": "0"}
	if _, err := ConfigFromEnv(func(k string) string { return env[k] }); err == nil {
		t.Fatal("MAX_SESSIONS=0 accepted")
	}
	env = map[string]string{"IDLE_TIMEOUT": "soon"}
	if _, err := ConfigFromEnv(func(k string) string { return env[k] }); err == nil {
		t.Fatal("bad IDLE_TIMEOUT accepted")
	}
}
```

Note on `TestShutdownWarnsOpenSessions`: `ssh.Server.Shutdown` waits for clients to hang up. A real `ssh` client exits when its session ends; the Go test client keeps the TCP connection, so the test asserts the session ends instead of waiting for `Shutdown` to return.

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./internal/server/`
Expected: FAIL — `undefined: Config`, `undefined: New`, `undefined: ConfigFromEnv`.

- [ ] **Step 4: Implement config, limits and the registry**

Create `internal/server/config.go`:

```go
package server

import (
	"fmt"
	"strconv"
	"time"
)

// Config is everything the server reads from the environment.
type Config struct {
	ListenAddr  string
	HostKeyPath string
	PublicHost  string // shown in the "ssh -t <host>" hint
	MaxSessions int
	PerIP       int
	IdleTimeout time.Duration
	MaxSession  time.Duration
}

// ConfigFromEnv reads the env vars listed in the spec, with defaults.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	c := Config{
		ListenAddr:  or(getenv("LISTEN_ADDR"), ":2222"),
		HostKeyPath: or(getenv("HOST_KEY_PATH"), "/data/ssh_host_ed25519"),
		PublicHost:  or(getenv("PUBLIC_HOST"), "term.kudayyurter.dev"),
		PerIP:       5,
	}
	var err error
	if c.MaxSessions, err = strconv.Atoi(or(getenv("MAX_SESSIONS"), "100")); err != nil || c.MaxSessions < 1 {
		return Config{}, fmt.Errorf("MAX_SESSIONS: must be a positive integer")
	}
	if c.IdleTimeout, err = time.ParseDuration(or(getenv("IDLE_TIMEOUT"), "10m")); err != nil {
		return Config{}, fmt.Errorf("IDLE_TIMEOUT: %w", err)
	}
	if c.MaxSession, err = time.ParseDuration(or(getenv("MAX_SESSION"), "30m")); err != nil {
		return Config{}, fmt.Errorf("MAX_SESSION: %w", err)
	}
	return c, nil
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
```

Create `internal/server/limits.go`:

```go
package server

import (
	"net"
	"sync"

	"charm.land/ssh"
	"charm.land/wish/v2"
)

// limiter caps concurrent sessions overall and per remote IP.
type limiter struct {
	mu         sync.Mutex
	max, perIP int
	total      int
	byIP       map[string]int
}

func newLimiter(max, perIP int) *limiter {
	return &limiter{max: max, perIP: perIP, byIP: map[string]int{}}
}

func (l *limiter) acquire(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.total >= l.max || l.byIP[ip] >= l.perIP {
		return false
	}
	l.total++
	l.byIP[ip]++
	return true
}

func (l *limiter) release(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.total--
	if l.byIP[ip]--; l.byIP[ip] <= 0 {
		delete(l.byIP, ip)
	}
}

func (l *limiter) middleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ip := remoteIP(s)
			if !l.acquire(ip) {
				wish.Println(s, "server busy, try again shortly")
				_ = s.Exit(1)
				return
			}
			defer l.release(ip)
			next(s)
		}
	}
}

func remoteIP(s ssh.Session) string {
	host, _, err := net.SplitHostPort(s.RemoteAddr().String())
	if err != nil {
		return s.RemoteAddr().String()
	}
	return host
}
```

Create `internal/server/registry.go`:

```go
package server

import (
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
)

// registry tracks running programs so shutdown can tell each one.
type registry struct {
	mu       sync.Mutex
	programs map[ssh.Session]*tea.Program
}

func newRegistry() *registry { return &registry{programs: map[ssh.Session]*tea.Program{}} }

func (r *registry) add(s ssh.Session, p *tea.Program) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.programs[s] = p
}

func (r *registry) broadcast(msg tea.Msg) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		p.Send(msg)
	}
}

func (r *registry) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.programs)
}

// middleware forgets a session's program once the session ends. It must sit
// outside (later in the list than) the Bubble Tea middleware.
func (r *registry) middleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			defer func() {
				r.mu.Lock()
				delete(r.programs, s)
				r.mu.Unlock()
			}()
			next(s)
		}
	}
}
```

- [ ] **Step 5: Implement the server**

Create `internal/server/server.go`:

```go
// Package server wires the SSH server: no authentication, per-session Bubble
// Tea programs for terminals, one-shot command mode without one, limits,
// logging and graceful shutdown.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	bm "charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/ratelimiter"
	"charm.land/wish/v2/recover"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/time/rate"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/boot"
	"github.com/namelessmonarch0/ssh-portfolio/internal/shell"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
	"github.com/namelessmonarch0/ssh-portfolio/internal/ui"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

// Server is the SSH server plus what it needs to shut down cleanly.
type Server struct {
	SSH      *ssh.Server
	cfg      Config
	fs       *vfs.FS
	profile  content.Profile
	log      *slog.Logger
	programs *registry
	projects int
}

// New builds the server. The host key at cfg.HostKeyPath is created on first
// start and reused afterwards so returning visitors never see a key warning.
func New(cfg Config, fsys *vfs.FS, profile content.Profile, log *slog.Logger) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.HostKeyPath), 0o700); err != nil {
		return nil, fmt.Errorf("host key directory: %w", err)
	}
	s := &Server{cfg: cfg, fs: fsys, profile: profile, log: log, programs: newRegistry()}
	if n, err := fsys.Resolve(vfs.Home, "projects"); err == nil {
		s.projects = len(n.Children())
	}

	// wish applies middlewares in list order, each wrapping the previous
	// one, so the LAST entry runs FIRST for every session.
	srv, err := wish.NewServer(
		wish.WithAddress(cfg.ListenAddr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithIdleTimeout(cfg.IdleTimeout),
		wish.WithMaxTimeout(cfg.MaxSession),
		wish.WithMiddleware(
			s.execMiddleware(),                         // sessions without a terminal
			bm.MiddlewareWithProgramHandler(s.program), // sessions with one
			s.programs.middleware(),
			newLimiter(cfg.MaxSessions, cfg.PerIP).middleware(),
			ratelimiter.Middleware(ratelimiter.NewRateLimiter(rate.Every(time.Second), 10, 10_000)),
			s.logMiddleware(),
			recover.Middleware(), // outermost: a panic ends one session, not the server
		),
	)
	if err != nil {
		return nil, fmt.Errorf("ssh server: %w", err)
	}
	s.SSH = srv
	return s, nil
}

// Shutdown warns every open session, stops accepting, and waits for sessions
// to close until ctx expires.
func (s *Server) Shutdown(ctx context.Context) error {
	s.programs.broadcast(ui.ShutdownMsg{})
	return s.SSH.Shutdown(ctx)
}

func (s *Server) welcome() string {
	return style.Bold.Render(s.profile.Name) + "\n" + style.Muted.Render(s.profile.Tagline) + "\n"
}

// program builds the Bubble Tea program for a session with a terminal, or
// returns nil (handing the session to execMiddleware) when there is none.
func (s *Server) program(sess ssh.Session) *tea.Program {
	pty, _, ok := sess.Pty()
	if !ok {
		return nil
	}
	env := append(sess.Environ(), "TERM="+pty.Term)
	plain := pty.Term == "dumb" || colorprofile.Env(env) < colorprofile.ANSI
	ip := remoteIP(sess)

	sh := shell.New(s.fs, s.profile, shell.Options{Width: pty.Window.Width, Interactive: true})
	m := ui.New(sh, ui.Options{
		Width:     pty.Window.Width,
		Height:    pty.Window.Height,
		Plain:     plain,
		Boot:      boot.Options{Tagline: s.profile.Tagline, Projects: s.projects},
		Welcome:   s.welcome(),
		OnCommand: func(line string) { s.log.Info("command", "remote", ip, "line", line) },
	})
	p := tea.NewProgram(m, bm.MakeOptions(sess)...)
	s.programs.add(sess, p)
	return p
}

// execMiddleware serves `ssh host <command>` and `ssh -T host`: plain text,
// no animation, the command's exit code as the session's exit status.
func (s *Server) execMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			if _, _, ok := sess.Pty(); ok {
				next(sess) // the interactive program already ran
				return
			}
			sh := shell.New(s.fs, s.profile, shell.Options{Width: 80})
			var res shell.Result
			if raw := sess.RawCommand(); raw != "" {
				s.log.Info("command", "remote", remoteIP(sess), "line", raw, "exec", true)
				res = sh.Run(raw)
			} else {
				res = sh.Run("cat about.md")
				res.Output += "\n\nFor the interactive version: ssh -t " + s.cfg.PublicHost
			}
			if out := ansi.Strip(res.Output); out != "" {
				wish.Println(sess, out)
			}
			_ = sess.Exit(res.Code)
			next(sess)
		}
	}
}

func (s *Server) logMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			start := time.Now()
			pty, _, isPty := sess.Pty()
			s.log.Info("connect", "remote", remoteIP(sess), "pty", isPty, "term", pty.Term,
				"width", pty.Window.Width, "height", pty.Window.Height)
			next(sess)
			s.log.Info("disconnect", "remote", remoteIP(sess), "duration", time.Since(start).Round(time.Second).String())
		}
	}
}
```

- [ ] **Step 6: Run the tests (three times, with the race detector)**

```bash
go mod tidy
go vet ./... && go test -race -count=3 ./internal/server/
```

Expected: PASS every time.

- [ ] **Step 7: Add the entry point**

Create `cmd/server/main.go`:

```go
// Command server runs the SSH portfolio.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/ssh"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/server"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := server.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	fsys, err := vfs.New(content.Files)
	if err != nil {
		return err
	}
	profile, err := content.LoadProfile()
	if err != nil {
		return err
	}
	srv, err := server.New(cfg, fsys, profile, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddr)
		errc <- srv.SSH.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err := srv.SSH.Close(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}
```

- [ ] **Step 8: Try it for real**

```bash
LISTEN_ADDR=127.0.0.1:2222 HOST_KEY_PATH=.data/host_ed25519 PUBLIC_HOST=localhost go run ./cmd/server &
SSHO="-p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PubkeyAuthentication=no"
ssh $SSHO localhost                # boot animation, then the shell; try ls, cat work/cummins.md, neofetch, exit
ssh $SSHO localhost ls -l work     # plain text
ssh $SSHO localhost nope; echo $?  # 127
kill %1                            # logs "shutting down" and exits
```

(In fish or zsh, write the options out instead of using `$SSHO`.) Check by eye: the boot animation is centered and ends in the shell; after `cat work/cummins.md` in a short window the prompt is still visible and complete; text can be selected and copied; the mouse wheel scrolls back through earlier output.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum internal/server cmd/server
git commit -m "feat: add SSH server with limits, exec mode and graceful shutdown

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 10: Container image, CI, and README

**Files:**
- Create: `Dockerfile`, `.dockerignore`, `.github/workflows/ci.yml`, `README.md`

**Interfaces:**
- Consumes: `cmd/server` (Task 9) and its env vars.
- Produces: image entrypoint `/ssh-portfolio`, listening on `:2222`, host key at `/data/ssh_host_ed25519`, running as distroless `nonroot` (uid 65532). CI job `test` (Task 11 adds `deploy` after it).

- [ ] **Step 1: Write the Dockerfile and .dockerignore**

Create `Dockerfile`:

```dockerfile
# Build a static binary, then ship it alone on a distroless base (~15 MB).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ssh-portfolio ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/ssh-portfolio /ssh-portfolio
ENV LISTEN_ADDR=:2222 HOST_KEY_PATH=/data/ssh_host_ed25519
EXPOSE 2222
VOLUME /data
USER nonroot:nonroot
ENTRYPOINT ["/ssh-portfolio"]
```

Create `.dockerignore`:

```gitignore
.git
.data
deploy/keys
docs
*.test
```

- [ ] **Step 2: Build and run the image hardened, as production will**

```bash
docker build -t ssh-portfolio:dev .
docker images ssh-portfolio:dev --format '{{.Size}}'   # about 30 MB
mkdir -p .data/docker
docker run -d --name ssh-portfolio-test --user "$(id -u):$(id -g)" --read-only --cap-drop ALL \
  --security-opt no-new-privileges -p 127.0.0.1:2299:2222 -v "$PWD/.data/docker:/data" ssh-portfolio:dev
ssh -p 2299 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PubkeyAuthentication=no localhost whoami
ls .data/docker                     # ssh_host_ed25519  ssh_host_ed25519.pub
docker stop -t 15 ssh-portfolio-test && docker logs ssh-portfolio-test | tail -1   # "shutting down"
docker rm ssh-portfolio-test
```

Expected: `guest — visiting Kuday Yurter · CS student · data science intern at Cummins`. (`--user` is only needed locally because `.data/docker` belongs to your user; on the server the volume is owned by uid 65532.)

- [ ] **Step 3: Add CI**

Create `.github/workflows/ci.yml`:

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }
      - run: go vet ./...
      - run: go test -race ./...
      - run: docker build -t ssh-portfolio:ci .
```

- [ ] **Step 4: Add the README**

Create `README.md`:

````markdown
# ssh-portfolio

Kuday Yurter's portfolio, over SSH:

```sh
ssh term.kudayyurter.dev
```

A pixel-name boot animation, then a small read-only shell: `ls`, `cd`, `cat`,
`tree`, `neofetch`, `help`. One-shot commands work too:
`ssh term.kudayyurter.dev cat contact.md`.

## Editing content

Everything visitors can read is in [`content/`](content/): each markdown file
is a file in the visitor's home directory, and its front matter drives
`ls -l` and the `cat` header.

```yaml
---
title: Cummins            # required
summary: Data Science Intern   # required; one line, shown by ls -l
date: May 2026 — Present  # optional
stack: Python · SQL       # optional
link: https://…           # optional; used by `open`
order: 1                  # optional; listing order, lowest first
---
```

Write links as bare URLs (`https://…`) or bare emails, not `[text](url)`,
so they print once and stay clickable. Name, tagline and the `neofetch`
card come from [`content/profile.yaml`](content/profile.yaml).
`go test ./content` checks front matter and that no city or state slipped in.

## Run locally

```sh
LISTEN_ADDR=:2222 HOST_KEY_PATH=.data/host_ed25519 go run ./cmd/server
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost
```

## Test

```sh
go test ./...
go test ./internal/boot -update   # after an intended animation change; review testdata/
```

## Deploy

Pushing to `main` runs the tests, builds the image, and ships it to the
Lightsail box over SSH (see `.github/workflows/ci.yml`).

| Thing | Where |
|---|---|
| Server | Lightsail `ssh-portfolio` (us-east-2), static IP `ssh-portfolio-ip` |
| Visitors | port 22 → container port 2222 |
| Admin login | `ssh -p 2200 ubuntu@term.kudayyurter.dev` |
| Logs | `sudo journalctl -u ssh-portfolio -f` |
| Host key | `/var/lib/ssh-portfolio/` (back it up; losing it warns returning visitors) |
| First-time setup | `deploy/lightsail.sh` |

Environment variables: `LISTEN_ADDR` (`:2222`), `HOST_KEY_PATH`
(`/data/ssh_host_ed25519`), `MAX_SESSIONS` (`100`), `IDLE_TIMEOUT` (`10m`),
`MAX_SESSION` (`30m`), `PUBLIC_HOST` (`term.kudayyurter.dev`).
````

- [ ] **Step 5: Verify everything once more**

Run: `test -z "$(gofmt -l .)" && go vet ./... && go test -race ./...`
Expected: all packages `ok`.

- [ ] **Step 6: Commit**

```bash
git add Dockerfile .dockerignore .github README.md
git commit -m "build: add container image, CI and README

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

---

### Task 11: Provisioning, deploy pipeline, and go-live

**Files:**
- Create: `deploy/cloud-init.sh`, `deploy/lightsail.sh`
- Modify: `.github/workflows/ci.yml` (append a `deploy` job)

**Interfaces:**
- Consumes: the image from Task 10.
- Produces: Lightsail instance `ssh-portfolio` (us-east-2) with static IP `ssh-portfolio-ip`; admin sshd on 2200 (key-only); systemd unit `ssh-portfolio` running `ssh-portfolio:current` with `-p 22:2222`; `deploy` user whose key may only run `/usr/local/bin/ssh-portfolio-deploy <sha>`; GitHub secrets `DEPLOY_HOST`, `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS`; DNS `A term.kudayyurter.dev`.

**Lockout safety:** first boot adds sshd on 2200 and only then drops 22; if 2200 does not come up it reverts and stops. Lightsail's browser SSH console uses port 22, so after the move it reaches the portfolio, not the box — keep your key for port 2200.

- [ ] **Step 1: Write the first-boot script**

Create `deploy/cloud-init.sh`:

```bash
#!/bin/bash
# First-boot setup for the Lightsail box (Ubuntu 24.04). lightsail.sh
# replaces __DEPLOY_PUBKEY__ before passing this file as user data.
# Log: /var/log/cloud-init-output.log
set -euxo pipefail

# 1. Move admin SSH from 22 to 2200 so the portfolio can own port 22.
#    Ubuntu 24.04 starts sshd from ssh.socket, whose ports are generated from
#    sshd_config, so reload units and restart the socket. If 2200 does not
#    come up, undo the change and stop: port 22 stays with sshd.
cat >/etc/ssh/sshd_config.d/10-portfolio.conf <<'EOF'
Port 2200
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
systemctl daemon-reload
systemctl restart ssh.socket
for _ in $(seq 1 20); do
  ss -ltn | grep -q ':2200 ' && break
  sleep 1
done
if ! ss -ltn | grep -q ':2200 '; then
  rm /etc/ssh/sshd_config.d/10-portfolio.conf
  systemctl daemon-reload
  systemctl restart ssh.socket
  echo "sshd did not come up on 2200; left on 22" >&2
  exit 1
fi

# 2. Docker.
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
systemctl enable --now docker

# 3. Host key volume, owned by distroless "nonroot" (uid 65532).
install -d -o 65532 -g 65532 -m 700 /var/lib/ssh-portfolio

# 4. The service. It starts on the first deploy, once an image exists.
cat >/etc/systemd/system/ssh-portfolio.service <<'EOF'
[Unit]
Description=SSH portfolio
After=docker.service ssh.socket
Requires=docker.service

[Service]
ExecStartPre=-/usr/bin/docker rm -f ssh-portfolio
ExecStart=/usr/bin/docker run --name ssh-portfolio --rm \
  -p 22:2222 -v /var/lib/ssh-portfolio:/data \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  --memory 256m --pids-limit 256 \
  ssh-portfolio:current
ExecStop=/usr/bin/docker stop -t 15 ssh-portfolio
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable ssh-portfolio

# 5. Deploy user: its key can only run the receiver below, which loads an
#    image from stdin, tags it current and restarts the service.
useradd --create-home --shell /bin/sh deploy
usermod -aG docker deploy
echo 'deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart ssh-portfolio' >/etc/sudoers.d/deploy
chmod 440 /etc/sudoers.d/deploy

cat >/usr/local/bin/ssh-portfolio-deploy <<'EOF'
#!/bin/sh
# Forced command for the deploy key. Usage (from CI):
#   docker save ssh-portfolio:<sha> | gzip | ssh -p 2200 deploy@host <sha>
set -eu
tag="${SSH_ORIGINAL_COMMAND:-}"
case "$tag" in
  "" | *[!0-9a-f]*) echo "expected a commit sha" >&2; exit 1 ;;
esac
gunzip | docker load
docker tag "ssh-portfolio:$tag" ssh-portfolio:current
sudo /usr/bin/systemctl restart ssh-portfolio
docker image prune -af --filter "until=168h" >/dev/null
echo "deployed $tag"
EOF
chmod 755 /usr/local/bin/ssh-portfolio-deploy

install -d -o deploy -g deploy -m 700 /home/deploy/.ssh
echo 'command="/usr/local/bin/ssh-portfolio-deploy",restrict __DEPLOY_PUBKEY__' >/home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
```

- [ ] **Step 2: Write the provisioning script**

Create `deploy/lightsail.sh`:

```bash
#!/usr/bin/env bash
# Creates the Lightsail box for the SSH portfolio. Run once from the repo root
# after `aws login`. Safe to re-run: existing resources are left alone.
#
#   ADMIN_PUBKEY=~/.ssh/id_ed25519.pub deploy/lightsail.sh
set -euo pipefail

NAME=ssh-portfolio
REGION=${REGION:-us-east-2}
AZ=${AZ:-${REGION}a}
ADMIN_PUBKEY=${ADMIN_PUBKEY:-$HOME/.ssh/id_ed25519.pub}
DEPLOY_KEY=deploy/keys/deploy_ed25519
export AWS_REGION=$REGION

# Deploy key for GitHub Actions (private half becomes a repo secret).
if [[ ! -f $DEPLOY_KEY ]]; then
  mkdir -p deploy/keys
  ssh-keygen -t ed25519 -N '' -C "ssh-portfolio deploy" -f "$DEPLOY_KEY"
fi

# Admin key pair (your own key) so you can log in on port 2200.
if ! aws lightsail get-key-pair --key-pair-name "$NAME-admin" >/dev/null 2>&1; then
  aws lightsail import-key-pair --key-pair-name "$NAME-admin" \
    --public-key-base64 "$(cat "$ADMIN_PUBKEY")"
fi

if ! aws lightsail get-instance --instance-name "$NAME" >/dev/null 2>&1; then
  # Cheapest Linux bundle that still includes a public IPv4 address.
  BUNDLE=$(aws lightsail get-bundles --query \
    "sort_by(bundles[?isActive && contains(supportedPlatforms, 'LINUX_UNIX') && !contains(bundleId, 'ipv6')], &price)[0].bundleId" \
    --output text)
  echo "bundle: $BUNDLE"

  USER_DATA=$(sed "s|__DEPLOY_PUBKEY__|$(cat "$DEPLOY_KEY.pub")|" deploy/cloud-init.sh)
  aws lightsail create-instances --instance-names "$NAME" \
    --availability-zone "$AZ" --blueprint-id ubuntu_24_04 --bundle-id "$BUNDLE" \
    --key-pair-name "$NAME-admin" --user-data "$USER_DATA"
fi

echo "waiting for the instance to run..."
until [[ $(aws lightsail get-instance-state --instance-name "$NAME" --query state.name --output text) == running ]]; do
  sleep 5
done

if ! aws lightsail get-static-ip --static-ip-name "$NAME-ip" >/dev/null 2>&1; then
  aws lightsail allocate-static-ip --static-ip-name "$NAME-ip"
  aws lightsail attach-static-ip --static-ip-name "$NAME-ip" --instance-name "$NAME"
fi

# 22: visitors. 2200: admin + deploys (key-only; GitHub runner IPs change).
aws lightsail put-instance-public-ports --instance-name "$NAME" --port-infos \
  'fromPort=22,toPort=22,protocol=TCP,cidrs=0.0.0.0/0' \
  'fromPort=2200,toPort=2200,protocol=TCP,cidrs=0.0.0.0/0'

IP=$(aws lightsail get-static-ip --static-ip-name "$NAME-ip" --query staticIp.ipAddress --output text)
cat <<EOF

Static IP: $IP
Next:
  1. Wait ~3 minutes for first-boot setup, then: ssh -p 2200 ubuntu@$IP 'tail -5 /var/log/cloud-init-output.log'
  2. DNS: A record  term.kudayyurter.dev -> $IP
  3. GitHub secrets: DEPLOY_HOST=$IP, DEPLOY_SSH_KEY=<contents of $DEPLOY_KEY>,
     DEPLOY_KNOWN_HOSTS=\$(ssh-keyscan -p 2200 $IP)
EOF
```

- [ ] **Step 3: Lint both scripts**

```bash
chmod +x deploy/*.sh
bash -n deploy/cloud-init.sh && bash -n deploy/lightsail.sh
docker run --rm -v "$PWD/deploy:/mnt:ro" koalaman/shellcheck:stable /mnt/cloud-init.sh /mnt/lightsail.sh
```

Expected: no output from shellcheck.

- [ ] **Step 4: Append the deploy job to `.github/workflows/ci.yml`** (under `jobs:`, after `test`)

```yaml
  deploy:
    needs: test
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    concurrency: deploy
    steps:
      - uses: actions/checkout@v5
      - run: docker build -t "ssh-portfolio:${GITHUB_SHA}" .
      - name: Ship the image to Lightsail
        env:
          DEPLOY_HOST: ${{ secrets.DEPLOY_HOST }}
          DEPLOY_SSH_KEY: ${{ secrets.DEPLOY_SSH_KEY }}
          DEPLOY_KNOWN_HOSTS: ${{ secrets.DEPLOY_KNOWN_HOSTS }}
        run: |
          install -d -m 700 ~/.ssh
          printf '%s\n' "$DEPLOY_SSH_KEY" > ~/.ssh/deploy
          chmod 600 ~/.ssh/deploy
          printf '%s\n' "$DEPLOY_KNOWN_HOSTS" > ~/.ssh/known_hosts
          docker save "ssh-portfolio:${GITHUB_SHA}" | gzip \
            | ssh -i ~/.ssh/deploy -p 2200 "deploy@${DEPLOY_HOST}" "${GITHUB_SHA}"
      - name: Smoke test
        env:
          DEPLOY_HOST: ${{ secrets.DEPLOY_HOST }}
        run: |
          sleep 3
          ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
            "guest@${DEPLOY_HOST}" whoami | grep -q '^guest'
```

- [ ] **Step 5: Commit**

```bash
git add deploy .github/workflows/ci.yml
git commit -m "ci: add Lightsail provisioning and deploy pipeline

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01SD2QvXzfZ3CZD6nW15tB18"
```

- [ ] **Step 6: STOP — ask the user** before any of the following outward-facing steps. Ask whether the GitHub repo should be public or private, and confirm the subdomain `term.kudayyurter.dev`.

- [ ] **Step 7: Create the GitHub repo and push** (after approval)

```bash
gh repo create namelessmonarch0/ssh-portfolio --public --source . --push   # or --private, per the user
```

Expected: the `test` job passes; `deploy` fails (no secrets yet) — that is fine for now.

- [ ] **Step 8: Provision the box** (after approval; the user runs `! aws login` first if the session expired)

```bash
ADMIN_PUBKEY=~/.ssh/id_ed25519.pub deploy/lightsail.sh
```

Wait ~3 minutes, then:

```bash
ssh -p 2200 ubuntu@<IP> 'tail -5 /var/log/cloud-init-output.log; ss -ltn | grep -E ":(22|2200) "; systemctl is-enabled ssh-portfolio'
```

Expected: cloud-init finished without error; sshd listens on 2200 only; `enabled`.

- [ ] **Step 9: Add DNS** (after approval): an `A` record `term` → `<IP>` on `kudayyurter.dev` in Vercel DNS (dashboard → Domains → kudayyurter.dev → DNS Records, or `npx vercel dns add kudayyurter.dev term A <IP>`). Check with `dig +short term.kudayyurter.dev`.

- [ ] **Step 10: Add the deploy secrets** (after approval)

```bash
gh secret set DEPLOY_HOST --body "<IP>"
gh secret set DEPLOY_SSH_KEY < deploy/keys/deploy_ed25519
ssh-keyscan -p 2200 <IP> 2>/dev/null | gh secret set DEPLOY_KNOWN_HOSTS
```

- [ ] **Step 11: First deploy and verification**

```bash
gh run rerun --failed $(gh run list --workflow ci --limit 1 --json databaseId --jq '.[0].databaseId')
gh run watch
ssh term.kudayyurter.dev                              # full experience
ssh term.kudayyurter.dev cat contact.md               # plain text
ssh -p 2200 ubuntu@term.kudayyurter.dev 'sudo journalctl -u ssh-portfolio -n 20'
```

Expected: both jobs green; the boot animation and shell work from your own terminal; logs show your **public** IP as `remote` (if they show `172.17.0.1`, Docker is proxying and per-IP limits would lump everyone together — switch the unit to `--network host` with `LISTEN_ADDR=:22` and `--cap-add NET_BIND_SERVICE`, then re-verify). Reconnecting after a second deploy must not show a host-key warning.

- [ ] **Step 12: Back up the host key** (it identifies the server to returning visitors)

```bash
ssh -p 2200 ubuntu@term.kudayyurter.dev 'sudo cat /var/lib/ssh-portfolio/ssh_host_ed25519' > deploy/keys/host_ed25519.backup
chmod 600 deploy/keys/host_ed25519.backup
```

`deploy/keys/` is git-ignored.
