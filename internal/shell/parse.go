package shell

import (
	"errors"
	"fmt"
	"strings"
)

var errUnsupported = errors.New("pipes and redirects aren't supported here")

// sanitize drops control characters so nothing a visitor types can smuggle
// terminal escape sequences into output. Tabs become spaces.
func sanitize(line string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, line)
}

// split turns a command line into words, honoring single and double quotes.
// Shell operators are rejected because nothing here can pipe or redirect.
func split(line string) ([]string, error) {
	var (
		words  []string
		cur    strings.Builder
		inWord bool
		quote  rune
	)
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		case strings.ContainsRune("|<>;&", r):
			return nil, errUnsupported
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unexpected EOF while looking for matching `%c'", quote)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// parseFlags separates single-letter flags (which may be combined, like -la)
// from operands. Letters not in allowed are an error in GNU wording.
func parseFlags(cmd string, args []string, allowed string) (map[rune]bool, []string, error) {
	set := map[rune]bool{}
	var operands []string
	for _, a := range args {
		if len(a) > 1 && a[0] == '-' {
			for _, r := range a[1:] {
				if !strings.ContainsRune(allowed, r) {
					return nil, nil, fmt.Errorf("%s: invalid option -- '%c'", cmd, r)
				}
				set[r] = true
			}
			continue
		}
		operands = append(operands, a)
	}
	return set, operands, nil
}
