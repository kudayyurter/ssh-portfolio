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
			s.execMiddleware(), // sessions without a terminal
			s.teaMiddleware(),  // sessions with one
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

// maxWidth and maxHeight cap the window size a client can claim. Every frame
// allocates width×height cells, so an absurd size is a memory attack.
const maxWidth, maxHeight = 512, 256

func clampWindow(w, h int) (int, int) {
	return min(max(w, 0), maxWidth), min(max(h, 0), maxHeight)
}

// teaMiddleware runs a Bubble Tea program for sessions with a terminal and
// hands the rest to the next handler. It is wish's bubbletea middleware with
// window sizes clamped before the program, and its renderer, see them.
func (s *Server) teaMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			pty, windows, ok := sess.Pty()
			if !ok {
				next(sess)
				return
			}
			p := s.program(sess, pty)
			ctx, cancel := context.WithCancel(sess.Context())
			go func() {
				for {
					select {
					case <-ctx.Done():
						p.Quit()
						return
					case w := <-windows:
						width, height := clampWindow(w.Width, w.Height)
						p.Send(tea.WindowSizeMsg{Width: width, Height: height})
					}
				}
			}()
			if _, err := p.Run(); err != nil {
				s.log.Error("program exited", "remote", remoteIP(sess), "err", err)
			}
			p.Kill()
			cancel()
			next(sess)
		}
	}
}

// program builds the Bubble Tea program for a session with a terminal.
func (s *Server) program(sess ssh.Session, pty ssh.Pty) *tea.Program {
	env := append(sess.Environ(), "TERM="+pty.Term)
	plain := pty.Term == "dumb" || colorprofile.Env(env) < colorprofile.ANSI
	ip := remoteIP(sess)
	width, height := clampWindow(pty.Window.Width, pty.Window.Height)

	sh := shell.New(s.fs, s.profile, shell.Options{Width: width, Interactive: true})
	m := ui.New(sh, ui.Options{
		Width:     width,
		Height:    height,
		Plain:     plain,
		Boot:      boot.Options{Tagline: s.profile.Tagline, Projects: s.projects},
		Welcome:   s.welcome(),
		OnCommand: func(line string) { s.log.Info("command", "remote", ip, "line", line) },
	})
	// MakeOptions wires the session's input and output; the later
	// WithWindowSize overrides the unclamped size it sets.
	p := tea.NewProgram(m, append(bm.MakeOptions(sess), tea.WithWindowSize(width, height))...)
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
