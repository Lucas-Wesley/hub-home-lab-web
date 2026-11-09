#!/usr/bin/env bash
# Publica o commit atual de origin/main no homelab.
# Uso: ./deploy.sh [--sem-testes]
set -euo pipefail

cd "$(dirname "$0")"

DEPLOY_HOST="${DEPLOY_HOST:-lucas@192.168.1.20}"
DEPLOY_KEY="${DEPLOY_KEY:-$HOME/.ssh/id_ed25519_homelab}"
APP_URL="${APP_URL:-https://192.168.1.20:8449}"
CA_CERT="${CA_CERT:-../certs/homelab-ca.crt}"

falhar() { echo "ERRO: $*" >&2; exit 1; }

git fetch -q origin main
SHA=$(git rev-parse HEAD)
[ "$SHA" = "$(git rev-parse origin/main)" ] \
  || falhar "o commit atual não é origin/main. Faça o push para a main antes de publicar."
[ -z "$(git status --porcelain)" ] \
  || echo "AVISO: há alterações locais; elas não serão publicadas."

if [ "${1:-}" != "--sem-testes" ]; then
  echo "==> Rodando os testes"
  make test
fi

echo "==> Publicando ${SHA:0:7}"
ssh -i "$DEPLOY_KEY" -o IdentitiesOnly=yes -o BatchMode=yes "$DEPLOY_HOST" \
  'bash -s --' "$SHA" < scripts/remote-deploy.sh

echo "==> Conferindo a versão no ar"
HEALTH=$(curl -fsS --max-time 15 --cacert "$CA_CERT" "$APP_URL/api/health") \
  || falhar "a API não respondeu em $APP_URL."
case "$HEALTH" in
  *"\"version\":\"$SHA\""*) ;;
  *) falhar "a versão no ar não é a publicada: $HEALTH" ;;
esac

# No ar, confere que o PWA responde e que o painel lê o disco e o Docker de verdade.
echo "==> Conferindo o PWA e o painel"
curl -fsS --max-time 15 --cacert "$CA_CERT" "$APP_URL/" | grep -q '<div id="app">' \
  || falhar "o PWA não respondeu em $APP_URL."
ARM=$(curl -fsS --max-time 60 --cacert "$CA_CERT" "$APP_URL/api/armazenamento") \
  || falhar "a API de armazenamento não respondeu."
case "$ARM" in
  *'"docker":{'*) ;;
  *) echo "AVISO: o painel não leu o Docker: $(echo "$ARM" | grep -o '"docker_erro":"[^"]*"')" ;;
esac

echo "==> Publicado: ${SHA:0:7} em $APP_URL"
