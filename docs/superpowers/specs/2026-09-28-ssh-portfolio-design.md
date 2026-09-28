# SSH Portfolio — Design

**Date:** 2026-09-28
**Status:** Approved in conversation, awaiting written-spec review
**Repo:** `~/DEV/ssh-portfolio` → `github.com/namelessmonarch0/ssh-portfolio`

## 1. Goal

Anyone can run `ssh term.kudayyurter.dev` and land in a terminal version of
Kuday's portfolio. It opens with a centered loading animation (pixel name
assembling from particles), then becomes something that looks and behaves like
a real shell: visitors explore with `ls`, `cd`, `cat`, `tree`, etc. and read
styled content about work, projects, stack, and contact.

It is a companion to kudayyurter.dev (the Next.js site on Vercel), aimed at
recruiters and developers, and exists to show craft.

### What the user said vs. what we assumed

| Said by user | Assumed (confirmed during design) |
|---|---|
| SSH in, feels like a portfolio | Same content as the website |
| Shell with real commands, plus fancy startup and clean look | No login — anyone can connect, no key required |
| Central loading effect, then a shell | Read-only; nothing a visitor types is stored except logs |
| Go + Wish, AWS Lightsail, own markdown content | Copy follows the site's style: plain language, no locations, facts only from `Resume/knowledge_base.md` / current site copy |

### Success criteria

- `ssh term.kudayyurter.dev` from a stock macOS/Linux/Windows terminal shows the
  boot animation, then a working prompt, with no password or key prompt.
- Every file in the virtual filesystem is readable with `cat`, and all listed
  commands behave as specified in §4.
- `ssh term.kudayyurter.dev cat about.md` prints plain text and exits.
- One visitor cannot crash or slow the server for others.
- Pushing to `main` deploys to production after tests pass.
- Returning visitors never see a host-key-changed warning after redeploys.

### Non-goals (YAGNI)

Visitor accounts, guestbook, analytics dashboard, IPv6, pipes/redirects/globs,
linking from kudayyurter.dev (possible follow-up), a real shell of any kind.

## 2. Approach

One Go binary using Charm's libraries (latest v2 majors, verified available
2026-09-28):

- `charm.land/wish/v2` — the SSH server itself (the app *is* the server; no
  OpenSSH involved in serving visitors)
- `charm.land/bubbletea/v2` — per-session TUI runtime (synchronized output,
  cell-diff renderer)
- `charm.land/lipgloss/v2` — styling, per-session color profile
- `charm.land/glamour/v2` — markdown rendering with a custom monochrome theme
- `charm.land/bubbles/v2` — text input and viewport building blocks

Rejected: a real restricted shell in a container (security surface, heavy,
awkward animations); Node `ssh2` + Ink (weaker PTY/resize handling). herdr
(`ssh apply@join.herdr.dev`) was analyzed as a reference: OpenSSH + Rust
ratatui app, key-only auth, particle→braille logo animation. We borrow the
particle-convergence idea, not the stack.

## 3. Architecture

```
cmd/server/        main: env config, host key, Wish server, graceful shutdown
internal/vfs/      virtual filesystem built from embedded content/
internal/shell/    command-line parsing, command registry, completion, history
internal/boot/     boot animation: pure frame(t, width, height) function
internal/ui/       root Bubble Tea model: boot → shell state, resize, scrollback
internal/style/    palette, prompt, glamour theme, color-profile handling
internal/pixelfont/ 5×7 glyphs ported from Portfolio/src/lib/pixel-font.ts
content/           markdown files = the filesystem (go:embed)
deploy/            Dockerfile, lightsail.sh, cloud-init, deploy workflow
```

Unit boundaries:

- **vfs** knows nothing about terminals. API: `Resolve(cwd, path) (Node, error)`,
  `List(dir)`, `Read(file)`, `Tree(dir)`. Nodes carry parsed front matter.
- **shell** takes a line + session state (cwd, history) and returns output as a
  structured result (styled string + exit code + optional side effect such as
  `Clear`, `Exit`, `ReplayBoot`). It does not touch Bubble Tea, so it can run
  in both interactive and exec mode.
- **boot** is a pure function of elapsed time and window size returning a
  rendered frame string; the Bubble Tea wrapper only ticks and forwards keys.
- **ui** composes boot + shell, owns scrollback, input line, key bindings.

### Session flow

1. Visitor connects. Wish accepts with no authentication (all auth methods
   pass; no key is required and none is stored).
2. Middleware chain (outermost first): panic recovery → logging → rate limit →
   session limits → mode dispatch.
3. **PTY session:** a fresh Bubble Tea program starts in `boot` state; after the
   animation or any keypress it switches to `shell` state.
4. **No PTY + command** (`ssh host cat about.md`): run the command through
   `shell` with a plain (no color, no animation) renderer, print, exit with the
   command's exit code.
5. **No PTY + no command** (`ssh -T host`): print `about.md` plain plus a
   one-line hint (`ssh -t term.kudayyurter.dev for the interactive version`),
   exit 0.
6. `exit`, `logout`, `Ctrl-D` on an empty line close the session.

### Limits & logging

- Per-IP connection rate limit (token bucket, ~1/s, burst 10) and max 5
  concurrent sessions per IP.
- Global max 100 concurrent sessions; extra connections get a one-line
  "server busy, try again shortly" and close.
- Idle timeout 10 min, max session 30 min.
- Structured JSON logs (`log/slog`) to stdout: connect/disconnect, remote IP,
  terminal size, TERM, commands typed. No keys, no passwords.

### Configuration (env vars)

| Var | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:2222` | SSH listen address inside the container |
| `HOST_KEY_PATH` | `/data/ssh_host_ed25519` | Persistent host key; generated on first start if missing |
| `MAX_SESSIONS` | `100` | Global concurrent session cap |
| `IDLE_TIMEOUT` | `10m` | Idle disconnect |
| `MAX_SESSION` | `30m` | Hard session cap |

## 4. The shell

### Filesystem

Home is `~` = `/home/guest`. `/` contains only `home/guest` (so `cd /` works
and feels real). Content is seeded from the current `Portfolio/src/content/portfolio.ts`.

```
~
├── about.md                       intro + education
├── contact.md                     email, GitHub, LinkedIn, website
├── stack.md                       tech stack, best-known first
├── work/
│   ├── cummins.md                 role, dates, highlights + its 5 projects
│   ├── engrave-me-now.md          role, dates, highlights + sales & inventory system
│   ├── university-of-houston.md
│   └── ifixandrepair.md
└── projects/
    ├── kessler.md
    ├── dispatch.md
    ├── snake-game.md
    ├── clash-of-valor.md
    └── lumon-boot-splash.md
```

Each file has YAML front matter:

```yaml
title: Cummins            # required
summary: Data Science Intern   # required, one line; shown by ls -l
date: May 2026 — Present  # optional
stack: Databricks · Python · Power BI   # optional
link: https://...         # optional; used by `open`
order: 1                  # optional; listing order (lower first)
```

Directory listings sort by `order` when present (so `work/` is newest first),
otherwise alphabetically, directories first.

### Commands

| Command | Behavior |
|---|---|
| `ls [-l] [-a] [path]` | Lists entries; dirs bold with trailing `/`. `-l` shows a column per entry with `summary` (and `date`/`stack` if set). `-a` adds `.` and `..`. Flags combine (`-la`). |
| `cd [path]` | Changes directory; no arg → `~`. Supports `~`, `..`, `.`, `/`, `-` (previous dir). |
| `pwd` | Prints absolute path. |
| `tree [path]` | Box-drawing tree with dir/file counts footer. |
| `cat <file…>` | Renders markdown via glamour, wrapped to width. Front matter shown as a compact header (title, date, stack, link). Directory → `cat: work: Is a directory`. |
| `less`, `more` | Aliases for `cat`. |
| `open <file>` | Prints the file's `link` as an OSC 8 clickable hyperlink; no link → `open: <file>: no link`. |
| `whoami` | One-line intro. |
| `neofetch` | Card: small pixel name on the left; role, school, top 5 stack items, links on the right. |
| `help` | Grouped command list with one-line descriptions. |
| `clear` | Clears scrollback. `Ctrl-L` does the same. |
| `history` | Numbered list of this session's commands. |
| `echo <args…>` | Prints args joined by spaces. |
| `date` | Current UTC date/time in `date(1)` format. |
| `exit`, `logout` | Close the session. |
| `boot` | Replays the boot animation. |
| `sudo …` | `guest is not in the sudoers file. This incident will be reported.` |
| `rm`, `mv`, `cp`, `touch`, `mkdir` | `<cmd>: read-only file system (nice try)` |
| `vim`, `vi`, `nano`, `emacs` | `<cmd>: read-only file system — try cat <file>` |

- Unknown command → `<cmd>: command not found`, plus `did you mean <x>?` when
  a command is within edit distance 2.
- Bad path → `<cmd>: <path>: No such file or directory`; unknown flag →
  `<cmd>: invalid option -- 'x'`. Exit codes: 0 ok, 1 error, 127 not found.
- Parsing: whitespace split with single/double quote support. `|`, `>`, `<`,
  `&&`, `;` → `pipes and redirects aren't supported here`.

### Line editing

↑/↓ history, ←/→ cursor, Home/End, Tab completion (commands in first position,
paths after; common-prefix completion, second Tab lists candidates),
`Ctrl-C` cancels the line (prints `^C`), `Ctrl-L` clears, `Ctrl-D` on empty
line exits, mouse wheel and PgUp/PgDn scroll history of output.

### Look

Monochrome, matching the site tokens: background terminal default (dark
expected), text `#ffffff`, muted `#a3a3a3`, faint `#808080`, lines `#262626`.
Prompt `guest@kuday:~$ ` (path part updates with cwd). On first prompt a dim
hint line `try: ls · cat about.md · help` is shown; it disappears after the first
command. After boot, a short welcome: small name, one tagline line, the hint.

Color profile is detected per session. `TERM=dumb` or no-color profile → plain
ASCII output and no boot animation.

## 5. Boot animation

- **Font:** 5×7 glyphs ported verbatim from `Portfolio/src/lib/pixel-font.ts`.
  "KUDAY YURTER" is 68×7 pixels (4-column word gap, 1-column letter gap).
- **Rendering with half blocks** (`▀ ▄ █`) so pixels are square. Scale by
  terminal size:
  - width ≥ 140 → 2× (136 cols × 7 rows)
  - width ≥ 72 → 1× (68 cols × 4 rows)
  - width ≥ 40 → stacked KUDAY / YURTER (35 cols × 8 rows)
  - smaller than 40×12 → skip animation, go straight to the shell
- **Timeline** (~2.1 s, 30 fps, everything centered):
  1. 0–1.0 s — faint `·` particles start at deterministic pseudo-random points
     near the screen edges and travel curved paths (ease-out) to their target
     pixel.
  2. 0.4–1.4 s — as each particle lands its pixel steps `·` → `▪` → full
     block, ordered by the site's `index × 37 mod total` scatter.
  3. 0.2–1.8 s — below the name: tagline (muted), thin progress bar, and a
     status line cycling `mounting /home/guest` → `indexing 5 projects` →
     `starting shell` → `[ ok ] ready`.
  4. 1.8–2.1 s — brief hold, then clear into the shell welcome.
- Any key skips (the key is swallowed). `boot` replays. Resize mid-animation
  re-centers without restarting. Frames are deterministic for a given
  `(t, width, height)` — no randomness at runtime.

## 6. Deployment

- **Image:** multi-stage Dockerfile, `CGO_ENABLED=0` static binary on
  `gcr.io/distroless/static:nonroot` (~15 MB). Read-only root filesystem, one
  volume at `/data` for the host key. Listens on `2222`; host maps `22 → 2222`.
- **Server:** AWS Lightsail, region `us-east-2`, smallest IPv4 bundle
  (~$5/mo), Ubuntu 24.04, static IP attached. Provisioned by
  `deploy/lightsail.sh` (AWS CLI) with a cloud-init user-data script that:
  installs Docker; adds admin sshd on port 2200; and only after 2200 is
  confirmed listening, removes 22 from sshd so the container can bind it.
- **Firewall (Lightsail):** 22/tcp open to all; 2200/tcp open only to the
  admin's IP (passed to the script).
- **DNS:** `A term.kudayyurter.dev → <static IP>` in Vercel DNS.
- **CD:** GitHub Actions on push to `main`: `go vet` + `go test ./...` →
  build + push image to `ghcr.io/namelessmonarch0/ssh-portfolio` → SSH to the
  box on port 2200 with a dedicated deploy key (repo secret) → `docker pull` +
  restart via a systemd unit that runs the container with
  `--restart unless-stopped`.
- **Graceful shutdown:** on SIGTERM stop accepting, print
  `system going down for update, reconnect in a moment` to open sessions, wait
  up to 10 s, exit.

## 7. Error handling

- A panic in one session is recovered, logged with a stack trace, and closes
  only that session.
- All visitor input errors produce shell-style messages (§4), never panics.
- Missing/unreadable host key path → generate a new ed25519 key there; if the
  directory is not writable, fail fast at startup with a clear log message.
- Content is embedded and validated at startup (front matter parses, required
  fields present); invalid content fails startup, and the same check runs in
  tests so it never reaches deploy.

## 8. Testing

- **vfs:** path resolution table tests (`..`, `~`, `/`, `./x`, `-`, trailing
  slashes, missing paths, file-as-dir).
- **shell:** golden-file tests for each command's output (plain renderer) at
  fixed width; completion and "did you mean" tables; parser quoting and
  unsupported-operator cases.
- **boot:** golden frames at t = 0, 0.5, 1.0, 1.5, 2.1 s for 160×45, 100×30,
  60×24, 30×10.
- **Integration:** start the server on a random port; connect with
  `golang.org/x/crypto/ssh` requesting a PTY; assert boot output then prompt;
  send `cat about.md\r` and assert content; exec-mode test
  (`cat about.md` without PTY) asserts plain output and exit status 0; unknown
  command exits 127.
- **Content lint:** every file has required front matter; content must not
  contain US state names, a curated list of city names (at least Houston,
  Victoria, Columbus, Indiana-area and Texas-area cities from the knowledge
  base), or uppercase two-letter state abbreviations as whole words after a
  comma (e.g. `, TX`). Institution names ("University of Houston",
  "Texas A&M University–Victoria", "Houston City College") are removed before
  the check.
- **Local:** `go run ./cmd/server` then `ssh -p 2222 localhost`.
- CI runs all of the above on every push; deploy only runs if they pass.

## 9. Open items for implementation time

- Subdomain is `term.kudayyurter.dev` unless the user picks another before DNS.
- AWS CLI session must be refreshed (`aws login`) before provisioning.
- GitHub repo creation and the deploy-key secret need the user's go-ahead
  (outward-facing actions).
