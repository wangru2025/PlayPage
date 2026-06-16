#!/usr/bin/env bash
set -euo pipefail

APP_ROOT=/opt/ai-static-host
SRC_ROOT="$APP_ROOT/src/apps/web"
WEB_ROOT="$APP_ROOT/web"
TMP_ROOT="$APP_ROOT/tmp"

install -d -m 755 "$WEB_ROOT"

runuser -u ai-static-host -- bash -lc "cd '$SRC_ROOT' && npm ci"
runuser -u ai-static-host -- bash -lc "cd '$SRC_ROOT' && /usr/bin/node ./node_modules/next/dist/bin/next build"

rm -rf "$WEB_ROOT/.next"
install -d -o ai-static-host -g ai-static-host "$WEB_ROOT/.next"

cp -a "$SRC_ROOT/.next/standalone" "$WEB_ROOT/.next/"
cp -a "$SRC_ROOT/.next/static" "$WEB_ROOT/.next/"

# Standalone server serves static assets from .next/static relative to its working tree.
install -d -o ai-static-host -g ai-static-host "$WEB_ROOT/.next/standalone/.next"
rm -rf "$WEB_ROOT/.next/standalone/.next/static"
cp -a "$SRC_ROOT/.next/static" "$WEB_ROOT/.next/standalone/.next/"

chown -R ai-static-host:ai-static-host "$WEB_ROOT/.next"

systemctl restart ai-static-host-web.service
