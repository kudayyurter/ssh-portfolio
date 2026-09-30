// Package shell is the pretend shell: it turns a command line into styled
// output. It knows nothing about terminals or SSH, so the same code serves
// interactive sessions and one-shot `ssh host <command>` runs.
package shell

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kudayyurter/ssh-portfolio/content"
	"github.com/kudayyurter/ssh-portfolio/internal/style"
	"github.com/kudayyurter/ssh-portfolio/internal/vfs"
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

// maxHistory is how many lines a session remembers.
const maxHistory = 500

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
	return style.UserHost() + style.Muted.Render(":") + style.Path.Render(vfs.Display(s.cwd)) + style.Muted.Render("$") + " "
}

// Run executes one command line.
func (s *Session) Run(line string) Result {
	line = sanitize(line)
	if strings.TrimSpace(line) == "" {
		return Result{}
	}
	s.history = append(s.history, line)
	if over := len(s.history) - maxHistory; over > 0 {
		s.history = slices.Delete(s.history, 0, over)
	}
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
func fail(out string) Result { return Result{Output: style.Err.Render(out), Code: 1} }
