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
