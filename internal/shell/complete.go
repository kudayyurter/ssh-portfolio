package shell

import "strings"

// Complete extends the word being typed. line is the text before the cursor.
// It returns the new text and, when several names still match and nothing
// more can be added, the candidates to show (directories end in "/").
func (s *Session) Complete(line string) (string, []string) {
	i := strings.LastIndex(line, " ")
	word := line[i+1:]

	if strings.TrimSpace(line[:i+1]) == "" { // first word: a command name
		var names []string
		for _, c := range visibleCommands() {
			if strings.HasPrefix(c, word) {
				names = append(names, c)
			}
		}
		return extend(line, word, names, nil)
	}

	dir, base := "", word
	if j := strings.LastIndex(word, "/"); j >= 0 {
		dir, base = word[:j+1], word[j+1:]
	}
	n, err := s.fs.Resolve(s.cwd, dir)
	if err != nil || !n.Dir {
		return line, nil
	}
	var names []string
	dirs := map[string]bool{}
	for _, c := range n.Children() {
		if strings.HasPrefix(c.Name, base) {
			names = append(names, c.Name)
			dirs[c.Name] = c.Dir
		}
	}
	return extend(line, base, names, dirs)
}

// extend replaces the typed prefix with the longest completion. A single
// match is finished with "/" for a directory or a space otherwise.
func extend(line, typed string, names []string, dirs map[string]bool) (string, []string) {
	stem := line[:len(line)-len(typed)]
	switch len(names) {
	case 0:
		return line, nil
	case 1:
		if dirs[names[0]] {
			return stem + names[0] + "/", nil
		}
		return stem + names[0] + " ", nil
	}
	prefix := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	shown := make([]string, len(names))
	for i, n := range names {
		shown[i] = n
		if dirs[n] {
			shown[i] += "/"
		}
	}
	return stem + prefix, shown
}
