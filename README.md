# ssh-portfolio

Kuday Yurter's portfolio, served over SSH. There's nothing to install and no password to type:

```sh
ssh term.kudayyurter.dev
```

The name first appears as a pixel-font boot animation (press any key to skip it).
Then you get a home card with a menu: **About**, **Work**, **Projects**,
**Stack** and **Contact**.

| Key | Does |
|---|---|
| `↑` `↓` / `j` `k` | move, or scroll a page |
| `Enter` / `→` / `l` | open |
| `Esc` / `←` / `h` / `Backspace` | back |
| `PgUp` `PgDn` `Space` `g` `G` | jump by a page, to the top, to the bottom |
| `q` / `Ctrl+C` | quit |

The mouse wheel also scrolls in terminals that support alternate scroll mode.
In terminals with OSC 8 hyperlinks, you can open links with Ctrl+click (Cmd+click on macOS).
The contact icons come from a Nerd Font. Each icon has a text label next to it, so other fonts still read clearly.

## One-shot commands

Without a terminal, the server runs a small read-only shell and prints plain text.
It returns each command's exit status:

```sh
ssh term.kudayyurter.dev cat contact.md
ssh term.kudayyurter.dev ls -l work
ssh term.kudayyurter.dev help        # every command: cd, ls, tree, cat, open, neofetch, …
```

`ssh -T term.kudayyurter.dev` with no command prints `about.md`.

## Editing content

Everything visitors can read lives in [`content/`](content/). Each markdown
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

- A new file in `content/work/` or `content/projects/` shows up in its list
  automatically. The home menu is fixed in code (`menuPaths` in
  [`internal/tui/tui.go`](internal/tui/tui.go)), so you need to add any new top-level page there.
- Write links as bare URLs (`https://…`) or bare emails, not `[text](url)`.
  That way each one prints once and stays clickable.
- The name, tagline, links and `neofetch` card come from
  [`content/profile.yaml`](content/profile.yaml).
- `go test ./content` checks the front matter and makes sure no city or state
  slipped in.

## Run locally

This needs Go 1.27 or newer. Start the server on port 2222; it creates a host key under the git-ignored `.data/`:

```sh
LISTEN_ADDR=:2222 HOST_KEY_PATH=.data/host_ed25519 go run ./cmd/server
```

In another terminal:

```sh
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost
```

## Test

```sh
go test ./...
go test ./internal/boot -update   # after an intended animation change
go test ./internal/tui -update    # after an intended layout change
```

After `-update`, review the rewritten `testdata/*.golden` files before you commit. CI
also runs `gofmt`, `go vet`, the race detector and a Docker build.

## Deploy

When you push to `main`, [CI](.github/workflows/ci.yml) runs the tests, builds the image and
ships it to the Lightsail box over SSH. It then smoke-tests the live server.

| Thing | Where |
|---|---|
| Server | Lightsail `ssh-portfolio` (us-east-2), static IP `ssh-portfolio-ip` |
| Visitors | port 22 → container port 2222 |
| Admin login | `ssh -p 2200 ubuntu@term.kudayyurter.dev` |
| Logs | `sudo journalctl -u ssh-portfolio -f` |
| Host key | `/var/lib/ssh-portfolio/` (back it up; if it's lost, returning visitors get a host key warning) |
| First-time setup | [`deploy/lightsail.sh`](deploy/lightsail.sh) |

The server reads these environment variables (defaults in parentheses):

| Variable | Default |
|---|---|
| `LISTEN_ADDR` | `:2222` |
| `HOST_KEY_PATH` | `/data/ssh_host_ed25519` |
| `PUBLIC_HOST` | `term.kudayyurter.dev` |
| `MAX_SESSIONS` | `100` |
| `IDLE_TIMEOUT` | `10m` |
| `MAX_SESSION` | `30m` |

`PUBLIC_HOST` is the host named in the `ssh -t` hint that one-shot sessions print.
