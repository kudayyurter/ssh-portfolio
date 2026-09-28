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
