#!/usr/bin/env bash
# Tests the sshd port move in cloud-init.sh (section "# 1.") with stubbed
# systemctl and ss, so the lockout-safety logic runs without a real box.
#   bash deploy/cloud-init_test.sh
set -uo pipefail

here=$(cd "$(dirname "$0")" && pwd)
failures=0

# run_case NAME EXPECT_EXIT EXPECT_CONF(yes|no) RESTART_FAILS(0|1) PORTS_AFTER
run_case() {
  local name=$1 want_exit=$2 want_conf=$3 restart_fails=$4 ports_after=$5
  local tmp
  tmp=$(mktemp -d)
  mkdir "$tmp/bin"

  # systemctl: records calls; the first "restart ssh.socket" may fail.
  cat >"$tmp/bin/systemctl" <<EOF
#!/bin/bash
echo "\$*" >>"$tmp/calls"
if [[ "\$*" == "restart ssh.socket" && $restart_fails == 1 && ! -e "$tmp/restarted" ]]; then
  touch "$tmp/restarted"; exit 1
fi
[[ "\$*" == "restart ssh.socket" ]] && touch "$tmp/restarted"
exit 0
EOF
  # ss: before any restart sshd is on 22; after it, on \$ports_after.
  cat >"$tmp/bin/ss" <<EOF
#!/bin/bash
ports="22"
[[ -e "$tmp/restarted" ]] && ports="$ports_after"
for p in \$ports; do echo "LISTEN 0 128 0.0.0.0:\$p 0.0.0.0:*"; done
EOF
  printf '#!/bin/bash\n' >"$tmp/bin/sleep"
  chmod +x "$tmp/bin/"*

  # Section 1 only, with the config path moved into the temp dir.
  sed -n '/^# 1\./,/^# 2\./p' "$here/cloud-init.sh" \
    | sed "s|/etc/ssh/sshd_config.d/10-portfolio.conf|$tmp/conf|g" >"$tmp/section.sh"

  PATH="$tmp/bin:$PATH" bash -euo pipefail "$tmp/section.sh" >/dev/null 2>&1
  local got_exit=$?
  local got_conf=no
  [[ -e "$tmp/conf" ]] && got_conf=yes

  if [[ $got_exit != "$want_exit" || $got_conf != "$want_conf" ]]; then
    echo "FAIL $name: exit $got_exit (want $want_exit), config left: $got_conf (want $want_conf)"
    failures=$((failures + 1))
  else
    echo "ok   $name"
  fi
  rm -rf "$tmp"
}

run_case "moves to 2200"                       0 yes 0 "2200"
run_case "2200 never comes up: roll back"      1 no  0 "22"
run_case "socket restart fails: roll back"     1 no  1 "2200"
run_case "22 still held after move: roll back" 1 no  0 "22 2200"

exit $((failures > 0))
