#!/bin/bash
# First-boot setup for the Lightsail box (Ubuntu 24.04). user-data.sh fills
# in __DEPLOY_PUBKEY__ and wraps this file so Lightsail's /bin/sh runs it
# with bash.
# Log: /var/log/cloud-init-output.log
set -euxo pipefail

# 1. Move admin SSH from 22 to 2200 so the portfolio can own port 22.
#    Ubuntu 24.04 starts sshd from ssh.socket, whose ports are generated from
#    sshd_config, so reload units and restart the socket. Success means sshd
#    listens on 2200 and no longer on 22. Anything else (including a failed
#    restart) removes the change and stops: port 22 stays with sshd.
#    Tested by deploy/cloud-init_test.sh.
cat >/etc/ssh/sshd_config.d/10-portfolio.conf <<'EOF'
Port 2200
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
# Not "ss | grep -q": grep exits at the first match, ss then dies of SIGPIPE,
# and pipefail turns a match into a failure.
listening() { grep -q ":$1 " <<<"$(ss -ltn)"; }
moved() {
  for _ in $(seq 1 20); do
    if listening 2200 && ! listening 22; then return 0; fi
    sleep 1
  done
  return 1
}
if ! { systemctl daemon-reload && systemctl restart ssh.socket && moved; }; then
  rm -f /etc/ssh/sshd_config.d/10-portfolio.conf
  systemctl daemon-reload || true
  systemctl restart ssh.socket || true
  echo "sshd did not move to 2200; left on 22" >&2
  exit 1
fi

# 2. Docker.
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
systemctl enable --now docker

# 3. Host key volume, owned by distroless "nonroot" (uid 65532).
install -d -o 65532 -g 65532 -m 700 /var/lib/termfolio

# 4. The service. It starts on the first deploy, once an image exists.
cat >/etc/systemd/system/termfolio.service <<'EOF'
[Unit]
Description=SSH portfolio
After=docker.service ssh.socket
Requires=docker.service

[Service]
ExecStartPre=-/usr/bin/docker rm -f termfolio
ExecStart=/usr/bin/docker run --name termfolio --rm \
  -p 22:2222 -v /var/lib/termfolio:/data \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  --memory 256m --pids-limit 256 \
  termfolio:current
ExecStop=/usr/bin/docker stop -t 15 termfolio
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable termfolio

# 5. Deploy user: its key can only run the receiver below, which loads an
#    image from stdin, tags it current and restarts the service.
useradd --create-home --shell /bin/sh deploy
usermod -aG docker deploy
echo 'deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart termfolio' >/etc/sudoers.d/deploy
chmod 440 /etc/sudoers.d/deploy

cat >/usr/local/bin/termfolio-deploy <<'EOF'
#!/bin/sh
# Forced command for the deploy key. Usage (from CI):
#   docker save termfolio:<sha> | gzip | ssh -p 2200 deploy@host <sha>
set -eu
tag="${SSH_ORIGINAL_COMMAND:-}"
case "$tag" in
  "" | *[!0-9a-f]*) echo "expected a commit sha" >&2; exit 1 ;;
esac
gunzip | docker load
docker tag "termfolio:$tag" termfolio:current
sudo /usr/bin/systemctl restart termfolio
docker image prune -af --filter "until=168h" >/dev/null
echo "deployed $tag"
EOF
chmod 755 /usr/local/bin/termfolio-deploy

install -d -o deploy -g deploy -m 700 /home/deploy/.ssh
echo 'command="/usr/local/bin/termfolio-deploy",restrict __DEPLOY_PUBKEY__' >/home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
