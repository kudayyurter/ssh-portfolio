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
