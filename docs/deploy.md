# Testing and deploying

## Test

```sh
go test ./...
go test ./internal/boot -update   # after an intended animation change
go test ./internal/tui -update    # after an intended layout change
```

After `-update`, review the rewritten `testdata/*.golden` files before you
commit. CI also runs `gofmt`, `go vet`, the race detector, the cloud-init test
([`deploy/cloud-init_test.sh`](../deploy/cloud-init_test.sh)) and a Docker build.

## Deploy

When you push to `main`, [CI](../.github/workflows/ci.yml) runs the tests.
If the push changes what goes into the image (`go.mod`, `go.sum`, `cmd/`,
`internal/`, `content/`, `Dockerfile` or `.dockerignore`), it also builds the
image, ships it to the Lightsail box over SSH, and smoke-tests the live server
with `ssh guest@<host> whoami`. Other pushes, such as docs or the license,
skip the deploy.

| Thing | Where |
|---|---|
| Server | Lightsail `ssh-portfolio` (us-east-2), static IP `ssh-portfolio-ip` (names from before the rename; Lightsail cannot rename them) |
| Visitors | port 22 → container port 2222 |
| Admin login | `ssh -p 2200 ubuntu@term.kudayyurter.dev` |
| Logs | `sudo journalctl -u termfolio -f` |
| Host key | `/var/lib/termfolio/` (back it up; if it's lost, returning visitors get a host key warning) |
| First-time setup | [`deploy/lightsail.sh`](../deploy/lightsail.sh) |
| CI secrets | `DEPLOY_HOST`, `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS` |

## Configuration

The server reads these environment variables:

| Variable | Default | What it does |
|---|---|---|
| `LISTEN_ADDR` | `:2222` | Address the SSH server listens on |
| `HOST_KEY_PATH` | `/data/ssh_host_ed25519` | Host key; created on first start, then reused |
| `PUBLIC_HOST` | `term.kudayyurter.dev` | Host named in the `ssh -t` hint that one-shot sessions print |
| `MAX_SESSIONS` | `100` | Open sessions across all visitors |
| `IDLE_TIMEOUT` | `10m` | Disconnect after this long without input |
| `MAX_SESSION` | `30m` | Longest a single session may last |

Fixed in code ([`internal/server/config.go`](../internal/server/config.go)):
5 sessions and 10 raw connections per IP, and a 15 s handshake timeout.
