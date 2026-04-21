#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info() { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()   { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_error(){ echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

# =====================================================
# CLEANUP
# =====================================================
log_info "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      log_info "  Deleting: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done
log_ok "Cleanup completed"

# =====================================================
# CREAZIONE FUNZIONE DIRETTA (base — math.log(2))
# =====================================================
log_info "Creating LnHarmonic-base (direct, math.log)"
$SERVERLEDGE create \
  --function lnHarmonic-base \
  --memory 128 \
  --src ../../examples/Tesi/LnHarmonic/lnHarmonic.py \
  --runtime python310 \
  --handler lnHarmonic.handler \
  --output "y:Float"
log_ok "Function lnHarmonic-base created"

# =====================================================
# CREAZIONE VARIANTI TRAMITE --approximate
# Carica automaticamente variants/LnHarmonic/LnHarmonic.json
# Varianti registrate:
#   LnHarmonic          (base, N=1M)
#   LnHarmonic-n200000
#   LnHarmonic-n100000
#   LnHarmonic-n50000
#   LnHarmonic-n10000
#   LnHarmonic-n5000
#   LnHarmonic-n1000
# =====================================================
log_info "Creating LnHarmonic with all variants (--approximate)"
$SERVERLEDGE create \
  --function LnHarmonic \
  --memory 128 \
  --src ../../variants/LnHarmonic/lnHarmonic.py \
  --runtime python310 \
  --handler lnHarmonic.handler \
  --output "y:Float" \
  --approximate \
  --shareContainer
log_ok "All LnHarmonic variants created"
