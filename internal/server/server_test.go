package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
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

// A client can claim any window size; a huge one used to make every frame
// allocate width×height cells and could get the container OOM-killed.
func TestHugeWindowSizeIsClamped(t *testing.T) {
	checkPeakHeap(t, func(addr string) {
		_, stdin, out := interactive(t, addr, 1500, 1000)
		waitFor(t, out, "mounting")
		time.Sleep(300 * time.Millisecond) // several animation frames
		io.WriteString(stdin, "x")
		waitFor(t, out, "guest@kuday")
	})
}

func TestHugeResizeIsClamped(t *testing.T) {
	checkPeakHeap(t, func(addr string) {
		sess, _, out := interactive(t, addr, 100, 30)
		waitFor(t, out, "mounting")
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

// Printing while the alt screen is still up loses the text, so the welcome
// must be written after the terminal leaves the alt screen.
func TestWelcomeAppearsAfterBoot(t *testing.T) {
	_, addr := startServer(t, testConfig(t))
	_, stdin, out := interactive(t, addr, 100, 30)
	waitFor(t, out, "mounting")
	io.WriteString(stdin, "x")
	waitFor(t, out, "guest@kuday")
	time.Sleep(200 * time.Millisecond)
	raw := out.String()
	i := strings.LastIndex(raw, "\x1b[?1049l")
	if i < 0 {
		t.Fatal("never left the alt screen")
	}
	if after := ansi.Strip(raw[i:]); !strings.Contains(after, "Kuday Yurter") {
		t.Fatalf("welcome not printed after leaving the alt screen:\n%s", after)
	}
}
