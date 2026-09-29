# SSH Portfolio TUI — Design

**Date:** 2026-09-28
**Status:** Draft, awaiting review
**Branch:** `feat/tui` (compared against the shell on `main`; not deployed)
**Builds on:** `2026-09-28-ssh-portfolio-design.md`

## 1. Goal

A second interface for `ssh term.kudayyurter.dev`: a menu-driven TUI instead of
a shell clone. It lives on its own branch so it can be run locally and compared
with the shell before choosing one.

### What the user said vs. what we assumed

| Said by user | Assumed (confirmed during design) |
|---|---|
| A TUI on a different branch, to compare with the shell | Same entry point; compared by running each branch locally; nothing deploys until one is chosen |
| Probably easier for recruiters, and it will look better | Keyboard first (arrows, Enter, Esc, q); mouse is a bonus |
| Home card + drill-in layout | Keeps the boot animation, markdown content, tokyodark `style` and contact links with icons |
| Work/Projects: list, then detail | Every entry gets its own page |
| Shell only for `ssh host cmd` | `ssh host cat about.md` keeps working unchanged |

### Success criteria

- After the boot animation, a visitor lands on a home card and can reach every
  page with arrows and Enter only, without knowing any command.
- Contact and project links open with a plain Ctrl+click (Cmd+click on macOS)
  in terminals that support OSC 8 — no Shift needed.
- The mouse wheel scrolls lists and pages in terminals with alternate scroll
  support.
- `ssh term.kudayyurter.dev cat about.md` prints the same plain text as on `main`.
- All existing limits (connections, window size, idle timeout, shutdown) still hold.

### Non-goals (YAGNI)

Clickable menu items, search/filter, the shell as a menu item, content changes,
deploy changes.

## 2. Screens

### Boot

Unchanged: the particle animation plays; any key or paste skips it. Plain
(colorless) terminals and windows too small for it start at home.

### Home

Centered in the window:

```
  ██  ██ ██  ██     Kuday Yurter
  ██ ██   ████      CS student · data science intern at Cummins
  ████     ██
  ██ ██    ██        kudayyurter.dev    github
  ██  ██   ██        linkedin           email

                  ▸ About
                    Work
                    Projects
                    Stack
                    Contact

            ↑↓ move · ↵ open · q quit
```

- Monogram: "KY" in the pixel font at 2×, as in `neofetch`.
- Name in `style.Heading`, tagline in `style.Muted`.
- Links: every entry of `profile.yaml`'s `links`, each an OSC 8 hyperlink with
  its Nerd Font icon (the map moves from `server.go`), two per row.
- Menu, fixed in code: About → `~/about.md`, Work → `~/work/`, Projects →
  `~/projects/`, Stack → `~/stack.md`, Contact → `~/contact.md`. An entry whose
  path does not exist in the vfs is left out.
- Under 72 columns the monogram is hidden and everything stacks in one column,
  links one per row.

### List page (a directory)

```
 Work                                            esc back
 ─────────────────────────────────────────────────────────
 ▸ Cummins           Data Science Intern · May 2026 — Present
   Engrave Me Now    ...
```

- One row per child, in vfs order: title (the selected row's in
  `style.Heading`), then `summary · date` in `style.Muted`, truncated to the
  width.
- `▸` marks the selected row; Enter opens it. A directory with no entries
  shows "nothing here yet".

### Detail page (a file)

- Header: title in `style.Heading`; `date · stack` in `style.Muted` when present;
  `link` as a clickable `style.URL` line when present.
- Body: `style.Markdown(body, width)`; wraps at most 80 columns (existing rule).
  If rendering fails, the raw body is shown.
- Links in the body need no extra work: glamour already emits OSC 8
  hyperlinks for markdown links, bare URLs and email addresses (so Contact is
  clickable as is).
- Long pages scroll; the footer shows `↓ more` while there is more below, and
  the scroll percentage.

### Keys

| Action | Keys |
|---|---|
| Move / scroll | ↑ ↓, j k, PgUp PgDn, space, g G (top / bottom) |
| Open | Enter, →, l |
| Back | Esc, ←, h, Backspace (no-op on home) |
| Quit | q, Ctrl+C, from any page |

Each page's footer shows only the keys that apply there. Going back restores
the previous page's cursor and scroll position.

### Mouse

The mouse is not captured. When the TUI starts, the program sends
`\x1b[?1007h` (alternate scroll: the terminal turns wheel motion into ↑/↓ in
the alt screen) and sends `\x1b[?1007l` before quitting. Links are then handled
by the terminal itself, so Ctrl+click works without Shift. Clicking menu items
does nothing — the accepted trade-off.

### Edge cases

- Window under 20×8: only "make the window a bit bigger" is shown; the page
  comes back when the window grows.
- Resize: every page re-renders at the new width; cursors and scroll offsets
  are clamped.
- Server shutdown: "system going down for update, reconnect in a moment" as a
  bottom bar for 1.5 s, then the session quits (same timing as today).

## 3. Architecture

```
server ──► ui (one per visitor) ──► boot   (unchanged)
                     │
                     └────────────► tui ──► vfs · style · pixelfont · content.Profile
exec (`ssh host cmd`) ─► shell               (unchanged)
```

### `internal/tui` (new)

Pure UI state: no SSH, timers or I/O, so it is tested directly.

| File | Responsibility |
|---|---|
| `tui.go` | `Model`: a stack of pages. `New(fs, profile, Options)`, `Update(tea.KeyPressMsg) (Model, Result)`, `View(w, h int) string`. Owns the fixed menu. `Result` says whether to quit and which path was opened (for logging). |
| `home.go` | Home card rendering, link grid, link icons, menu cursor. |
| `list.go` | List page from a directory node's children. |
| `detail.go` | Detail page: header from `Meta`, markdown body. |
| `scroll.go` | Scroller shared by lists and details: offset, visible rows, clamp, `↓ more` / percent. |

### `internal/ui` (shrinks)

- Keeps: boot mode, window size, shutdown notice, plain start, alt screen.
- Removes: shell mode — prompt, line editing, history, completion, scrollback,
  mouse wheel handling.
- After boot, every key goes to `tui.Model`; `View` renders it with the cursor
  hidden and `MouseMode` off.
- Sends the alternate-scroll on/off sequences with `tea.Raw`
  (`tea.Sequence(tea.Raw(off), tea.Quit)` on quit).
- `Options`: `Welcome` and `OnCommand` are replaced by `OnOpen(path string)`.

### `internal/server`

- `program()` builds `tui.New(...)` instead of a shell session and passes
  `OnOpen` that logs `"open", "remote", ip, "path", path`.
- `welcome()` and its test assertions are removed (the home card replaces it);
  the link-icon map moves to `tui`.
- `execMiddleware` is untouched.

### `internal/shell`

Unchanged. Its interactive-only parts (prompt, completion, boot/clear/exit
actions) become unused on this branch; they stay so the comparison diff is
about the TUI and switching back is trivial.

## 4. Testing

Standard `go test ./...`; CI unchanged.

- **`internal/tui`**, with fixture content:
  - Navigation: home → Work → an entry → back → back lands on home with the
    cursor still on Work; q quits from every depth; Backspace on home is a no-op.
  - Every key in the table; `g`/`G`/PgUp/PgDn stay in bounds.
  - Home at 100×30 and 60×24: menu order, monogram only when wide, every
    profile link present as an OSC 8 hyperlink with its icon.
  - List rows show title and summary; empty directory shows "nothing here yet";
    missing menu paths are skipped.
  - Detail header fields; long body scrolls with the right marker; Contact's
    URLs and email come out as OSC 8 hyperlinks.
  - Resize clamps scroll; tiny-window message appears and goes away.
  - Golden screens (ANSI stripped) of home, a list and a detail page at 80×24
    in `internal/tui/testdata/`, updated with `-update` like the boot goldens.
- **`internal/ui`**: a key or paste during boot skips to home; plain start
  skips boot; alternate scroll on at start and off on quit; `MouseMode` off;
  shutdown bar then quit. Shell-mode tests are removed with the code.
- **`internal/server`** (real SSH sessions): home card after boot; `j`,
  Enter opens a page and logs it; `ssh host cat about.md` output unchanged.
