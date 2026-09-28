package content

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func TestFilesBuildAValidTree(t *testing.T) {
	fsys, err := vfs.New(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"about.md", "contact.md", "stack.md", "work/cummins.md", "projects/kessler.md"} {
		if _, err := fsys.Resolve(vfs.Home, p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	projects, _ := fsys.Resolve(vfs.Home, "projects")
	if n := len(projects.Children()); n != 5 {
		t.Errorf("projects has %d files, want 5", n)
	}
}

func TestProfileLoads(t *testing.T) {
	p, err := LoadProfile()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Stack) != 5 || len(p.Links) == 0 {
		t.Fatalf("profile = %+v", p)
	}
}

// Institution names are allowed; everything else that names a place is not.
var institutions = []string{
	"Texas A&M University–Victoria",
	"University of Houston",
	"Houston City College",
}

var places = regexp.MustCompile(`(?i)\b(alabama|alaska|arizona|arkansas|california|colorado|connecticut|delaware|florida|georgia|hawaii|idaho|illinois|indiana|iowa|kansas|kentucky|louisiana|maine|maryland|massachusetts|michigan|minnesota|mississippi|missouri|montana|nebraska|nevada|new hampshire|new jersey|new mexico|new york|north carolina|north dakota|ohio|oklahoma|oregon|pennsylvania|rhode island|south carolina|south dakota|tennessee|texas|utah|vermont|virginia|washington|west virginia|wisconsin|wyoming|houston|victoria|columbus|austin|dallas|san antonio|indianapolis)\b`)

var stateCodes = regexp.MustCompile(`,\s*(AL|AK|AZ|AR|CA|CO|CT|DE|FL|GA|HI|ID|IL|IN|IA|KS|KY|LA|ME|MD|MA|MI|MN|MS|MO|MT|NE|NV|NH|NJ|NM|NY|NC|ND|OH|OK|OR|PA|RI|SC|SD|TN|TX|UT|VT|VA|WA|WV|WI|WY)\b`)

// locations returns every place name or state code in text, ignoring institution names.
func locations(text string) []string {
	for _, inst := range institutions {
		text = strings.ReplaceAll(text, inst, "")
	}
	return append(places.FindAllString(text, -1), stateCodes.FindAllString(text, -1)...)
}

func TestNoLocations(t *testing.T) {
	check := func(name, text string) {
		if found := locations(text); len(found) > 0 {
			t.Errorf("%s mentions locations: %q", name, found)
		}
	}
	err := fs.WalkDir(Files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(Files, p)
		if err != nil {
			return err
		}
		check(p, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	check("profile.yaml", string(profileYAML))
}

func TestLocationLintCatchesPlaces(t *testing.T) {
	for _, bad := range []string{"Based in Houston.", "Columbus, IN", "Worked in Texas", "Remote, TX"} {
		if len(locations(bad)) == 0 {
			t.Errorf("lint missed %q", bad)
		}
	}
	if found := locations("IT Support at University of Houston, then Texas A&M University–Victoria."); len(found) > 0 {
		t.Errorf("lint flagged institution names: %q", found)
	}
}
