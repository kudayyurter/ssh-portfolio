# Editing content

Everything visitors can read lives in [`content/`](../content/). Each markdown
file is one page, and its front matter sets how the page is listed:

```yaml
---
title: Cummins                 # required
summary: Data Science Intern   # required; one line, shown in lists and ls -l
date: May 2026 — Present       # optional
stack: Python · SQL            # optional
link: https://…                # optional; top of the page, and `open`
order: 1                       # optional; listing order, lowest first
---
```

| File or folder | What it holds |
|---|---|
| `content/*.md` | Top-level pages: About, Stack, Contact |
| `content/work/` | One file per job; each shows up in the Work list automatically |
| `content/projects/` | One file per project; each shows up in the Projects list automatically |
| [`content/profile.yaml`](../content/profile.yaml) | Name, tagline, links, and the `neofetch` card |

- The home menu is fixed in code (`menuPaths` in
  [`internal/tui/tui.go`](../internal/tui/tui.go)), so add any new top-level
  page there too.
- Write links as bare URLs (`https://…`) or bare emails, not `[text](url)`.
  That way each one prints once and stays clickable.
- The markdown files are embedded into the binary
  ([`content/embed.go`](../content/embed.go)), so rebuild or redeploy after an edit.
- `go test ./content` checks the front matter and makes sure no city or state
  slipped in.
