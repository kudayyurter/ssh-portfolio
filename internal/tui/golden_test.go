package tui

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

// Golden screens make visual changes show up in review. After an intended
// change run: go test ./internal/tui -update, then look at testdata/.
func TestGoldenScreens(t *testing.T) {
	screens := map[string][]string{
		"home_80x24.golden":    {},
		"work_80x24.golden":    {"j", "enter"},
		"kessler_80x24.golden": {"j", "j", "enter", "enter"},
	}
	for name, keys := range screens {
		m := New(testFS(t, fixture), testProfile, 80, 24)
		m, _ = press(m, keys...)
		got := plain(m.View()) + "\n"
		path := filepath.Join("testdata", name)
		if *update {
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run with -update to create)", err)
		}
		if got != string(want) {
			t.Errorf("%s changed; run with -update if intended\ngot:\n%s", name, got)
		}
	}
}
