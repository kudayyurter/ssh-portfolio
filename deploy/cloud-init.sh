#!/bin/bash
# First-boot setup for the Lightsail box (Ubuntu 24.04). lightsail.sh
# replaces __DEPLOY_PUBKEY__ before passing this file as user data.
# Log: /var/log/cloud-init-output.log
set -euxo pipefail

# 1. Move admin SSH from 22 to 2200 so the portfolio can own port 22.
#    Ubuntu 24.04 starts sshd from ssh.socket, whose ports are generated from
#    sshd_config, so reload units and restart the socket. If 2200 does not
#    come up, undo the change and stop: port 22 stays with sshd.
cat >/etc/ssh/sshd_config.d/10-portfolio.conf <<'EOF'
Port 2200
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
systemctl daemon-reload
systemctl restart ssh.socket
for _ in $(seq 1 20); do
  ss -ltn | grep -q ':2200 ' && break
  sleep 1
done
if ! ss -ltn | grep -q ':2200 '; then
  rm /etc/ssh/sshd_config.d/10-portfolio.conf
  systemctl daemon-reload
  systemctl restart ssh.socket
  echo "sshd did not come up on 2200; left on 22" >&2
  exit 1
fi

# 2. Docker.
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
systemctl enable --now docker

# 3. Host key volume, owned by distroless "nonroot" (uid 65532).
install -d -o 65532 -g 65532 -m 700 /var/lib/ssh-portfolio

# 4. The service. It starts on the first deploy, once an image exists.
cat >/etc/systemd/system/ssh-portfolio.service <<'EOF'
[Unit]
Description=SSH portfolio
After=docker.service ssh.socket
Requires=docker.service

[Service]
ExecStartPre=-/usr/bin/docker rm -f ssh-portfolio
ExecStart=/usr/bin/docker run --name ssh-portfolio --rm \
  -p 22:2222 -v /var/lib/ssh-portfolio:/data \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  --memory 256m --pids-limit 256 \
  ssh-portfolio:current
ExecStop=/usr/bin/docker stop -t 15 ssh-portfolio
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable ssh-portfolio

# 5. Deploy user: its key can only run the receiver below, which loads an
#    image from stdin, tags it current and restarts the service.
useradd --create-home --shell /bin/sh deploy
usermod -aG docker deploy
echo 'deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart ssh-portfolio' >/etc/sudoers.d/deploy
chmod 440 /etc/sudoers.d/deploy

cat >/usr/local/bin/ssh-portfolio-deploy <<'EOF'
#!/bin/sh
# Forced command for the deploy key. Usage (from CI):
#   docker save ssh-portfolio:<sha> | gzip | ssh -p 2200 deploy@host <sha>
set -eu
tag="${SSH_ORIGINAL_COMMAND:-}"
case "$tag" in
  "" | *[!0-9a-f]*) echo "expected a commit sha" >&2; exit 1 ;;
esac
gunzip | docker load
docker tag "ssh-portfolio:$tag" ssh-portfolio:current
sudo /usr/bin/systemctl restart ssh-portfolio
docker image prune -af --filter "until=168h" >/dev/null
echo "deployed $tag"
EOF
chmod 755 /usr/local/bin/ssh-portfolio-deploy

install -d -o deploy -g deploy -m 700 /home/deploy/.ssh
echo 'command="/usr/local/bin/ssh-portfolio-deploy",restrict __DEPLOY_PUBKEY__' >/home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
