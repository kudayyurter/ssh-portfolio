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
