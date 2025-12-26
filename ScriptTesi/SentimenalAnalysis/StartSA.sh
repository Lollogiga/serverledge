#!/bin/bash
set -euo pipefail

# =====================================================
# COLORI PER I LOG
# =====================================================
BLUE="\e[34m"
GREEN="\e[32m"
YELLOW="\e[33m"
RED="\e[31m"
RESET="\e[0m"

log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

# =====================================================
# PERCORSO CLI SERVERLEDGE
# =====================================================
SERVERLEDGE="../../bin/serverledge-cli"

if [[ ! -x "$SERVERLEDGE" ]]; then
  log_error "serverledge-cli not found or not executable"
  exit 1
fi

# =====================================================
# CLEANUP FUNZIONI SERVERLEDGE
# =====================================================
# =====================================================
# CLEANUP FUNZIONI SERVERLEDGE (JSON multiline)
# =====================================================
log_info "Cleaning existing Serverledge functions"

$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      log_info "Deleting function: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done

log_ok "Serverledge functions cleanup completed"


# =====================================================
# CREAZIONE FUNZIONE SENTIMETALANALYSIS LIGHT
# =====================================================
log_info "Creating SentimentalAnalysis function (light, heavy)"

$SERVERLEDGE create \
  --function SALight \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SALight.py \
  --handler SALight.handler \
  --input "text:Text" \
  --output "label:Text,confidence:Float" \
  --memory 2048


log_ok "Function SAlight created"

# =====================================================
# CREAZIONE FUNZIONE SENTIMETALANALYSIS HEAVY
# =====================================================
$SERVERLEDGE create \
  --function SAHeavy \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SAHeavy.py \
  --handler SAHeavy.handler \
  --input "text:Text" \
  --output "label:Text,confidence:Float" \
  --memory 2048

log_ok "Function SAHeavy created"

# =====================================================
# DONE
# =====================================================
log_ok "Workflow completed successfully"
