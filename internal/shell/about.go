package shell

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/namelessmonarch0/ssh-portfolio/internal/pixelfont"
	"github.com/namelessmonarch0/ssh-portfolio/internal/style"
)

func init() {
	register("whoami", command{run: runWhoami, group: "About", usage: "whoami", about: "who you are, and whose portfolio this is"})
	register("neofetch", command{run: runNeofetch, group: "About", usage: "neofetch", about: "the whole portfolio on one card"})
	register("help", command{run: runHelp, group: "Utilities", usage: "help", about: "this list"})
}

func runWhoami(s *Session, _ []string) Result {
	return ok("guest " + style.Muted.Render("— visiting "+s.profile.Name+" · "+s.profile.Tagline))
}

// monogram is "KY" in the site's pixel font at 2×: 22 columns × 7 rows.
var monogram = pixelfont.HalfBlocks(pixelfont.Scale(pixelfont.MustText("KY"), 2))

func runNeofetch(s *Session, _ []string) Result {
	p := s.profile
	rows := [][2]string{
		{"Name", p.Name},
		{"Role", p.Role},
		{"School", p.School},
		{"Degree", p.Degree},
		{"Stack", strings.Join(p.Stack, " · ")},
	}
	for _, l := range p.Links {
		rows = append(rows, [2]string{l.Label, style.Link(l.URL, style.LinkLabel(l.URL))})
	}
	info := []string{style.Bold.Render("guest@kuday"), style.Rule.Render(strings.Repeat("─", 11))}
	for _, r := range rows {
		if r[1] != "" {
			info = append(info, style.Muted.Render(fmt.Sprintf("%-9s", r[0]))+r[1])
		}
	}
	card := strings.Join(info, "\n")
	if s.width >= 72 || s.width == 0 {
		card = lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(monogram, "\n"), "   ", card)
	}
	lines := strings.Split(card, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return ok(strings.Join(lines, "\n"))
}

var helpGroups = []string{"Moving around", "Reading", "About", "Utilities"}

func runHelp(_ *Session, _ []string) Result {
	var b strings.Builder
	for i, g := range helpGroups {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(style.Bold.Render(g))
		var names []string
		for name, c := range commands {
			if c.group == g {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			c := commands[name]
			fmt.Fprintf(&b, "\n  %-18s%s", c.usage, style.Muted.Render(c.about))
		}
	}
	b.WriteString("\n\n" + style.Faint.Render("Tab completes · ↑/↓ history · Ctrl-L clears · exit leaves"))
	return ok(b.String())
}
