#!/bin/bash
set -euo pipefail

# Definizione colori per l'output
BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }
log_error(){ echo -e "${RED}[ERROR]${RESET} $1" >&2; }

# Verifica presenza CLI
SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

# ---------------------------------------------------------
# PULIZIA
# ---------------------------------------------------------
log_info "Cleaning existing Serverledge functions"
# Nota: questo comando cancella TUTTE le funzioni registrate.
# Se vuoi cancellare solo quelle di VectorMagnitude, bisognerebbe filtrare con grep.
$SERVERLEDGE list 2>/dev/null | tr -d '[]",' | sed '/^$/d' | while read -r fn; do
  log_info "Deleting function: $fn"
  $SERVERLEDGE delete --function "$fn" || true
done
log_ok "Cleanup completed"

# ---------------------------------------------------------
# CREAZIONE FUNZIONI VECTOR MAGNITUDE
# ---------------------------------------------------------
log_info "Creating VectorMagnitude (Python / Python Light)"

# 1. Versione Standard (Math.hypot / Pitagora)
$SERVERLEDGE create \
  --function VectorMagnitude-py \
  --memory 256 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitude.py \
  --runtime python310 \
  --handler VectorMagnitude.handler \
  --input "x:Float,y:Float" \
  --output "magnitude:Float"

# 2. Versione Light (Alpha Max Plus Beta)
$SERVERLEDGE create \
  --function VectorMagnitudeLight-py \
  --memory 256 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitudeLight.py \
  --runtime python310 \
  --handler VectorMagnitudeLight.handler \
  --input "x:Float,y:Float" \
  --output "magnitude:Float"

log_ok "Workflow completed successfully"