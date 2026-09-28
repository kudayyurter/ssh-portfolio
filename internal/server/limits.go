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

func remoteIP(s ssh.Session) string { return hostOf(s.RemoteAddr()) }

// connCallback applies the limiter to raw TCP connections, before any SSH
// handshake work, so idle or half-open connections can't pile up. Returning
// nil makes the server close the connection.
func (l *limiter) connCallback(_ ssh.Context, conn net.Conn) net.Conn {
	ip := hostOf(conn.RemoteAddr())
	if !l.acquire(ip) {
		return nil
	}
	return &countedConn{Conn: conn, release: func() { l.release(ip) }}
}

// countedConn gives its limiter slot back when the server closes it.
type countedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *countedConn) Close() error {
	c.once.Do(c.release)
	return c.Conn.Close()
}

func hostOf(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
