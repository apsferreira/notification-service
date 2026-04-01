#!/usr/bin/env bash
# submit-whatsapp-templates.sh
# Submete os templates de WhatsApp Business para aprovação da Meta.
#
# Uso:
#   export WHATSAPP_TOKEN="seu_token_aqui"
#   ./scripts/submit-whatsapp-templates.sh
#
#   Para submeter um template específico:
#   ./scripts/submit-whatsapp-templates.sh otp_code
#
# Pré-requisitos:
#   - jq instalado (brew install jq)
#   - WHATSAPP_TOKEN exportado no ambiente
#   - WABA ID: 951693297335694

set -euo pipefail

WABA_ID="951693297335694"
API_VERSION="v20.0"
BASE_URL="https://graph.facebook.com/${API_VERSION}/${WABA_ID}/message_templates"
TEMPLATES_FILE="$(dirname "$0")/../docs/whatsapp-templates.json"

# Verificações iniciais
if [[ -z "${WHATSAPP_TOKEN:-}" ]]; then
  echo "ERRO: variável WHATSAPP_TOKEN não definida."
  echo "Execute: export WHATSAPP_TOKEN='seu_token_aqui'"
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "ERRO: jq não encontrado. Instale com: brew install jq"
  exit 1
fi

if [[ ! -f "$TEMPLATES_FILE" ]]; then
  echo "ERRO: arquivo de templates não encontrado em: $TEMPLATES_FILE"
  exit 1
fi

FILTER_NAME="${1:-}"

submit_template() {
  local name="$1"
  local category="$2"
  local language="$3"
  local components="$4"

  local payload
  payload=$(jq -n \
    --arg name "$name" \
    --arg category "$category" \
    --arg language "$language" \
    --argjson components "$components" \
    '{name: $name, category: $category, language: $language, components: $components}')

  echo ""
  echo "Submetendo template: $name ($category / $language)"
  echo "---"

  local response
  response=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL" \
    -H "Authorization: Bearer $WHATSAPP_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload")

  local body
  local status
  status=$(echo "$response" | tail -n 1)
  body=$(echo "$response" | sed '$d')

  if [[ "$status" == "200" ]] || [[ "$status" == "201" ]]; then
    local template_id
    template_id=$(echo "$body" | jq -r '.id // "N/A"')
    echo "OK — ID: $template_id | Status HTTP: $status"
  else
    echo "ERRO — Status HTTP: $status"
    echo "Resposta: $body"
  fi
}

total=$(jq '.templates | length' "$TEMPLATES_FILE")
submitted=0
skipped=0
errors=0

echo "=============================="
echo " IIT — Submissão de Templates"
echo "=============================="
echo "WABA ID  : $WABA_ID"
echo "API      : $BASE_URL"
echo "Arquivo  : $TEMPLATES_FILE"
echo "Templates: $total encontrados"
if [[ -n "$FILTER_NAME" ]]; then
  echo "Filtro   : $FILTER_NAME"
fi
echo "=============================="

for i in $(seq 0 $((total - 1))); do
  name=$(jq -r ".templates[$i].name" "$TEMPLATES_FILE")
  category=$(jq -r ".templates[$i].category" "$TEMPLATES_FILE")
  language=$(jq -r ".templates[$i].language" "$TEMPLATES_FILE")
  components=$(jq -c ".templates[$i].components" "$TEMPLATES_FILE")

  if [[ -n "$FILTER_NAME" ]] && [[ "$name" != "$FILTER_NAME" ]]; then
    skipped=$((skipped + 1))
    continue
  fi

  submit_template "$name" "$category" "$language" "$components"
  submitted=$((submitted + 1))

  # Aguarda 1s entre submissões para evitar rate limit da API
  if [[ $i -lt $((total - 1)) ]] && [[ -z "$FILTER_NAME" ]]; then
    sleep 1
  fi
done

echo ""
echo "=============================="
echo "Concluído."
echo "Submetidos : $submitted"
echo "Ignorados  : $skipped"
echo "=============================="

# Verifica status dos templates após submissão
echo ""
echo "Status atual dos templates na conta:"
echo "---"
curl -s \
  "https://graph.facebook.com/${API_VERSION}/${WABA_ID}/message_templates?fields=name,status,category,language&limit=20" \
  -H "Authorization: Bearer $WHATSAPP_TOKEN" \
  | jq -r '.data[] | "\(.name) | \(.category) | \(.language) | \(.status)"' 2>/dev/null \
  || echo "(não foi possível listar — verifique o token)"
