# ssh-portfolio

Kuday Yurter's portfolio, over SSH:

```sh
ssh term.kudayyurter.dev
```

A pixel-name boot animation, then a small read-only shell: `ls`, `cd`, `cat`,
`tree`, `neofetch`, `help`. One-shot commands work too:
`ssh term.kudayyurter.dev cat contact.md`.

## Editing content

Everything visitors can read is in [`content/`](content/): each markdown file
is a file in the visitor's home directory, and its front matter drives
`ls -l` and the `cat` header.

```yaml
---
title: Cummins            # required
summary: Data Science Intern   # required; one line, shown by ls -l
date: May 2026 — Present  # optional
stack: Python · SQL       # optional
link: https://…           # optional; used by `open`
order: 1                  # optional; listing order, lowest first
---
```

Write links as bare URLs (`https://…`) or bare emails, not `[text](url)`,
so they print once and stay clickable. Name, tagline and the `neofetch`
card come from [`content/profile.yaml`](content/profile.yaml).
`go test ./content` checks front matter and that no city or state slipped in.

## Run locally

```sh
LISTEN_ADDR=:2222 HOST_KEY_PATH=.data/host_ed25519 go run ./cmd/server
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost
```

## Test

```sh
go test ./...
go test ./internal/boot -update   # after an intended animation change; review testdata/
```

## Deploy

Pushing to `main` runs the tests, builds the image, and ships it to the
Lightsail box over SSH (see `.github/workflows/ci.yml`).

| Thing | Where |
|---|---|
| Server | Lightsail `ssh-portfolio` (us-east-2), static IP `ssh-portfolio-ip` |
| Visitors | port 22 → container port 2222 |
| Admin login | `ssh -p 2200 ubuntu@term.kudayyurter.dev` |
| Logs | `sudo journalctl -u ssh-portfolio -f` |
| Host key | `/var/lib/ssh-portfolio/` (back it up; losing it warns returning visitors) |
| First-time setup | `deploy/lightsail.sh` |

Environment variables: `LISTEN_ADDR` (`:2222`), `HOST_KEY_PATH`
(`/data/ssh_host_ed25519`), `MAX_SESSIONS` (`100`), `IDLE_TIMEOUT` (`10m`),
`MAX_SESSION` (`30m`), `PUBLIC_HOST` (`term.kudayyurter.dev`).
