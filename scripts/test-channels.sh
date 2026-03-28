#!/bin/bash
# Testa os canais do notification-service
# Uso: ./scripts/test-channels.sh [email] [phone]
#
# Exemplos:
#   ./scripts/test-channels.sh
#   ./scripts/test-channels.sh apsf88@gmail.com
#   ./scripts/test-channels.sh apsf88@gmail.com +5571999999999
#
# Variaveis de ambiente:
#   BASE_URL  - URL base do servico (default: http://localhost:3030)

set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:3030}
EMAIL=${1:-apsf88@gmail.com}
PHONE=${2:-""}

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; }

echo "=== Testando notification-service em $BASE_URL ==="
echo ""

# --- Health ---
echo -n "Health check: "
HEALTH=$(curl -sf --max-time 5 "$BASE_URL/health" 2>/dev/null || echo '{"status":"unreachable"}')
STATUS=$(echo "$HEALTH" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','?'))" 2>/dev/null || echo "?")
if [ "$STATUS" = "ok" ]; then
  ok "$STATUS"
else
  fail "$STATUS"
  echo "Resposta: $HEALTH"
  echo ""
  warn "Servico inacessivel em $BASE_URL. Verifique se esta rodando."
  echo "  docker compose up -d   # local"
  echo "  kubectl port-forward -n production svc/notification-service 3030:3030   # K3s"
  exit 1
fi
echo ""

# --- OTP via email ---
echo "--- OTP via email para $EMAIL ---"
RESP=$(curl -s --max-time 10 -X POST "$BASE_URL/otp/send" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"channel\":\"email\"}" 2>&1)
echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"
echo ""

# --- OTP via telegram ---
echo "--- OTP via telegram para $EMAIL ---"
RESP=$(curl -s --max-time 10 -X POST "$BASE_URL/otp/send" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"channel\":\"telegram\"}" 2>&1)
echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"
echo ""

# --- OTP via whatsapp (somente se phone informado) ---
if [ -n "$PHONE" ]; then
  echo "--- OTP via whatsapp para $PHONE ---"
  RESP=$(curl -s --max-time 10 -X POST "$BASE_URL/otp/send" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"phone\":\"$PHONE\",\"channel\":\"whatsapp\"}" 2>&1)
  echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"
  echo ""
else
  warn "Phone nao informado — pulando teste de WhatsApp"
  echo "  Uso: $0 $EMAIL +5571999999999"
  echo ""
fi

echo "Done."
