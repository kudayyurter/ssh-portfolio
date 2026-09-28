#!/usr/bin/env bash
# Prints the Lightsail user data for a deploy public key.
#
# Lightsail runs user data with /bin/sh (dash) and prepends its own lines, so
# a bash shebang is ignored. The user data therefore only writes cloud-init.sh
# to a file with a POSIX heredoc and runs it with bash.
#
#   deploy/user-data.sh deploy/keys/deploy_ed25519.pub
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
pubkey=$(cat "$1")

cat <<EOF
mkdir -p /var/lib/portfolio-init
cat >/var/lib/portfolio-init/setup.sh <<'PORTFOLIO_INIT'
$(sed "s|__DEPLOY_PUBKEY__|$pubkey|" "$here/cloud-init.sh")
PORTFOLIO_INIT
bash /var/lib/portfolio-init/setup.sh
EOF
