package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
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

		MaxConnsPerIP:    10,
		HandshakeTimeout: 15 * time.Second,
	}
}

func newServer(t *testing.T, cfg Config) *Server {
	return newServerLog(t, cfg, io.Discard)
}

// newServerLog is newServer with its log written to w.
func newServerLog(t *testing.T, cfg Config, w io.Writer) *Server {
	t.Helper()
	fsys, err := vfs.New(content.Files)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := content.LoadProfile()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(cfg, fsys, profile, slog.New(slog.NewTextHandler(w, nil)))
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
	waitForWithin(t, out, want, 5*time.Second)
}

// waitForBoot waits until the boot animation is on screen. It looks for the
// tagline, which stays up for the whole animation; status lines like
// "mounting" can be skipped entirely when the first frame is slow.
func waitForBoot(t *testing.T, out *syncBuffer, d time.Duration) {
	t.Helper()
	waitForWithin(t, out, "data science intern", d)
}

// waitForWithin is waitFor with a custom deadline, for sessions whose frames
// are slow to render under -race on small CI runners (512×256 windows).
func waitForWithin(t *testing.T, out *syncBuffer, want string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
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
	logs := &syncBuffer{}
	addr := testsession.Listen(t, newServerLog(t, testConfig(t), logs).SSH)
	s, stdin, out := interactive(t, addr, 100, 30)

	waitForBoot(t, out, 5*time.Second)
	io.WriteString(stdin, "x") // any key skips it
	waitFor(t, out, "Kuday Yurter")
	io.WriteString(stdin, "j")
	io.WriteString(stdin, "\r")
	waitFor(t, out, "Engrave Me Now") // a Work entry ("Cummins" is also in the tagline)
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(logs.String(), "msg=open") || !strings.Contains(logs.String(), "path=/home/guest/work") {
		if time.Now().After(deadline) {
			t.Fatalf("open not logged:\n%s", logs.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	io.WriteString(stdin, "h") // back to home
	io.WriteString(stdin, "G")
	io.WriteString(stdin, "\r") // Contact, the last menu entry
	waitFor(t, out, "kudayyurter@gmail.com")
	io.WriteString(stdin, "q")

	done := make(chan error, 1)
	go func() { done <- s.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("session ended with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session did not end after q")
	}
}

func TestSmallTerminalSkipsBoot(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	_, _, out := interactive(t, addr, 30, 10)
	waitFor(t, out, "Kuday Yurter")
	if strings.Contains(ansi.Strip(out.String()), "mounting") {
		t.Fatal("boot animation shown in a 30×10 terminal")
	}
}

func TestSessionLimit(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxSessions = 1
	_, addr := startServer(t, cfg)
	_, _, out := interactive(t, addr, 100, 30)
	waitForBoot(t, out, 5*time.Second)

	got, err := session(t, addr).Output("pwd")
	if exitStatus(err) != 1 || !strings.Contains(string(got), "server busy") {
		t.Fatalf("second session: %q exit %d", got, exitStatus(err))
	}
}

func TestShutdownWarnsOpenSessions(t *testing.T) {
	srv, addr := startServer(t, testConfig(t))
	sess, stdin, out := interactive(t, addr, 100, 30)
	waitForBoot(t, out, 5*time.Second)
	io.WriteString(stdin, "x")
	waitFor(t, out, "Kuday Yurter")
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
		c.IdleTimeout != 10*time.Minute || c.MaxSession != 30*time.Minute || c.PublicHost != "term.kudayyurter.dev" || c.PerIP != 5 ||
		c.MaxConnsPerIP != 10 || c.HandshakeTimeout != 15*time.Second {
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

// A client can claim any window size; a huge one used to make every frame
// allocate width×height cells and could get the container OOM-killed.
func TestHugeWindowSizeIsClamped(t *testing.T) {
	checkPeakHeap(t, func(addr string) {
		_, stdin, out := interactive(t, addr, 1500, 1000)
		waitForBoot(t, out, 30*time.Second)
		time.Sleep(300 * time.Millisecond) // several animation frames
		io.WriteString(stdin, "x")
		waitForWithin(t, out, "Kuday Yurter", 30*time.Second)
	})
}

func TestHugeResizeIsClamped(t *testing.T) {
	checkPeakHeap(t, func(addr string) {
		sess, _, out := interactive(t, addr, 100, 30)
		waitForBoot(t, out, 5*time.Second)
		if err := sess.WindowChange(1000, 1500); err != nil {
			t.Fatal(err)
		}
		time.Sleep(500 * time.Millisecond) // several animation frames at the new size
	})
}

// checkPeakHeap runs a session against a fresh server and fails if the heap
// grows past 200 MB while it runs.
func checkPeakHeap(t *testing.T, session func(addr string)) {
	t.Helper()
	_, addr := startServer(t, testConfig(t))
	var peak uint64
	stop := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		defer close(sampled)
		var ms runtime.MemStats
		for {
			select {
			case <-stop:
				return
			case <-time.After(10 * time.Millisecond):
				runtime.ReadMemStats(&ms)
				peak = max(peak, ms.HeapInuse)
			}
		}
	}()
	session(addr)
	close(stop)
	<-sampled
	if peak > 200<<20 {
		t.Fatalf("heap peaked at %d MB for one oversized window", peak>>20)
	}
}

// After the animation the home card shows the contact links, clickable.
func TestHomeShowsContactLinks(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	_, stdin, out := interactive(t, addr, 100, 30)
	waitForBoot(t, out, 5*time.Second)
	io.WriteString(stdin, "x")
	waitFor(t, out, "Kuday Yurter")
	for _, label := range []string{" web", " github", " linkedin", " email"} {
		waitFor(t, out, label)
	}
	for _, url := range []string{
		"https://kudayyurter.dev",
		"https://github.com/kudayyurter",
		"https://www.linkedin.com/in/kudayyurter/",
		"mailto:kudayyurter@gmail.com",
	} {
		if !strings.Contains(out.String(), "\x1b]8;;"+url+"\a") {
			t.Errorf("home has no hyperlink to %s", url)
		}
	}
	if strings.Contains(out.String(), "\x1b[?1049l") {
		t.Fatal("left the alt screen after the animation; the TUI should stay full screen")
	}
	if !strings.Contains(out.String(), "\x1b[?1007h") {
		t.Fatal("alternate scroll was not turned on")
	}
}

func TestExecRejectsLongCommands(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	out, err := session(t, addr).Output("echo " + strings.Repeat("a", 2000))
	if exitStatus(err) != 1 || !strings.Contains(string(out), "command too long") {
		t.Fatalf("long command: exit %d, %q", exitStatus(err), out[:min(len(out), 80)])
	}
}

// banner reads the SSH version line, or returns "" if the server hangs up.
func banner(t *testing.T, conn net.Conn) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, _ := conn.Read(buf)
	return string(buf[:n])
}

func TestConnectionsPerIPAreCapped(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxConnsPerIP = 2
	_, addr := startServer(t, cfg)
	for i := range 3 {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		got := banner(t, conn)
		if i < 2 && !strings.HasPrefix(got, "SSH-2.0") {
			t.Fatalf("connection %d: no banner (%q)", i+1, got)
		}
		if i == 2 && got != "" {
			t.Fatalf("third connection from one IP was accepted: %q", got)
		}
	}
}

func TestSilentConnectionsTimeOut(t *testing.T) {
	cfg := testConfig(t)
	cfg.HandshakeTimeout = 300 * time.Millisecond
	_, addr := startServer(t, cfg)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	banner(t, conn)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("connection still open after the handshake timeout: %v", err)
	}
}
