#!/usr/bin/env bash
# Roda na VM, chamado pelo deploy.sh. Se o app não ficar saudável, volta ao commit anterior.
set -euo pipefail

SHA="$1"
REPO=https://github.com/Lucas-Wesley/hub-home-lab-web.git
APP_DIR=/opt/apps/hub
STATE=/opt/apps/hub.deployed
COMPOSE=(docker compose -f compose.yaml -f compose.prod.yaml)

[ -d "$APP_DIR/.git" ] || git clone -q "$REPO" "$APP_DIR"
cd "$APP_DIR"
git fetch -q origin
git checkout -q --detach "$SHA"

PREV=$(cat "$STATE" 2>/dev/null || true)

# O token do Proxmox fica em /opt/apps/hub.env, fora da repo.
[ -f /opt/apps/hub.env ] && cp /opt/apps/hub.env "$APP_DIR/.env"
[ -f "$APP_DIR/.env" ] || echo "AVISO: sem /opt/apps/hub.env; o painel abre sem os números do Proxmox." >&2
mkdir -p /srv/data/hub

export VERSION="$SHA"
"${COMPOSE[@]}" build --quiet

if ! "${COMPOSE[@]}" up -d --wait --wait-timeout 60; then
  echo "ERRO: ${SHA:0:7} não ficou saudável." >&2
  "${COMPOSE[@]}" logs --tail 20 >&2 || true
  if [ -n "$PREV" ]; then
    echo "==> Voltando para ${PREV:0:7}" >&2
    git checkout -q --detach "$PREV"
    VERSION="$PREV" "${COMPOSE[@]}" up -d --wait --wait-timeout 60
  fi
  exit 1
fi

echo "$SHA" > "$STATE"

# Mantém só as imagens do commit atual e do anterior
docker images hub --format '{{.Tag}}' \
  | { grep -vxF -e "$SHA" -e "${PREV:-$SHA}" || true; } \
  | xargs -r -I{} docker rmi -f "hub:{}" >/dev/null
docker image prune -f >/dev/null
