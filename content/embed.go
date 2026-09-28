// Package content holds the portfolio text. The markdown files are the
// visitor's home directory; profile.yaml holds the identity strings used by
// the boot screen, whoami and neofetch.
package content

import (
	"bytes"
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Files is the home directory: every markdown file under content/.
//
//go:embed *.md work/*.md projects/*.md
var Files embed.FS

//go:embed profile.yaml
var profileYAML []byte

// Link is a labeled URL.
type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// Profile is who the portfolio is about.
type Profile struct {
	Name    string   `yaml:"name"`
	Tagline string   `yaml:"tagline"`
	Role    string   `yaml:"role"`
	School  string   `yaml:"school"`
	Degree  string   `yaml:"degree"`
	Stack   []string `yaml:"stack"`
	Links   []Link   `yaml:"links"`
}

// LoadProfile parses profile.yaml.
func LoadProfile() (Profile, error) {
	var p Profile
	dec := yaml.NewDecoder(bytes.NewReader(profileYAML))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return Profile{}, fmt.Errorf("profile.yaml: %w", err)
	}
	if p.Name == "" || p.Tagline == "" {
		return Profile{}, fmt.Errorf("profile.yaml: name and tagline are required")
	}
	return p, nil
}
