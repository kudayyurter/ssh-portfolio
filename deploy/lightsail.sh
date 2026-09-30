#!/usr/bin/env bash
# Creates the Lightsail box for the SSH portfolio. Run once from the repo root
# after `aws login`. Safe to re-run: existing resources are left alone.
#
#   ADMIN_PUBKEY=~/.ssh/id_ed25519.pub deploy/lightsail.sh
set -euo pipefail

NAME=termfolio
REGION=${REGION:-us-east-2}
AZ=${AZ:-${REGION}a}
ADMIN_PUBKEY=${ADMIN_PUBKEY:-$HOME/.ssh/id_ed25519.pub}
DEPLOY_KEY=deploy/keys/deploy_ed25519
export AWS_REGION=$REGION

# Deploy key for GitHub Actions (private half becomes a repo secret).
if [[ ! -f $DEPLOY_KEY ]]; then
  mkdir -p deploy/keys
  ssh-keygen -t ed25519 -N '' -C "termfolio deploy" -f "$DEPLOY_KEY"
fi

# Admin key pair (your own key) so you can log in on port 2200.
if ! aws lightsail get-key-pair --key-pair-name "$NAME-admin" >/dev/null 2>&1; then
  aws lightsail import-key-pair --key-pair-name "$NAME-admin" \
    --public-key-base64 "$(cat "$ADMIN_PUBKEY")"
fi

if ! aws lightsail get-instance --instance-name "$NAME" >/dev/null 2>&1; then
  # Cheapest Linux bundle that still includes a public IPv4 address.
  BUNDLE=$(aws lightsail get-bundles --query \
    "sort_by(bundles[?isActive && contains(supportedPlatforms, 'LINUX_UNIX') && !contains(bundleId, 'ipv6')], &price)[0].bundleId" \
    --output text)
  echo "bundle: $BUNDLE"

  USER_DATA=$(deploy/user-data.sh "$DEPLOY_KEY.pub")
  aws lightsail create-instances --instance-names "$NAME" \
    --availability-zone "$AZ" --blueprint-id ubuntu_24_04 --bundle-id "$BUNDLE" \
    --key-pair-name "$NAME-admin" --user-data "$USER_DATA"
fi

echo "waiting for the instance to run..."
until [[ $(aws lightsail get-instance-state --instance-name "$NAME" --query state.name --output text) == running ]]; do
  sleep 5
done

if ! aws lightsail get-static-ip --static-ip-name "$NAME-ip" >/dev/null 2>&1; then
  aws lightsail allocate-static-ip --static-ip-name "$NAME-ip"
  aws lightsail attach-static-ip --static-ip-name "$NAME-ip" --instance-name "$NAME"
fi

# 22: visitors. 2200: admin + deploys (key-only; GitHub runner IPs change).
aws lightsail put-instance-public-ports --instance-name "$NAME" --port-infos \
  'fromPort=22,toPort=22,protocol=TCP,cidrs=0.0.0.0/0' \
  'fromPort=2200,toPort=2200,protocol=TCP,cidrs=0.0.0.0/0'

IP=$(aws lightsail get-static-ip --static-ip-name "$NAME-ip" --query staticIp.ipAddress --output text)
cat <<EOF

Static IP: $IP
Next:
  1. Wait ~3 minutes for first-boot setup, then: ssh -p 2200 ubuntu@$IP 'tail -5 /var/log/cloud-init-output.log'
  2. DNS: A record  term.kudayyurter.dev -> $IP
  3. GitHub secrets: DEPLOY_HOST=$IP, DEPLOY_SSH_KEY=<contents of $DEPLOY_KEY>,
     DEPLOY_KNOWN_HOSTS=\$(ssh-keyscan -p 2200 $IP)
EOF
