#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info() { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()   { echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { echo "serverledge-cli not found"; exit 1; }

# Invocazione diretta di ogni variante (senza variant-selection)
for VARIANT in LnHarmonic LnHarmonic-n200000 LnHarmonic-n100000 \
               LnHarmonic-n50000 LnHarmonic-n10000 \
               LnHarmonic-n5000 LnHarmonic-n1000; do
  log_info "Invoking ${VARIANT}"
  $SERVERLEDGE invoke \
    --function "${VARIANT}"
  log_ok "${VARIANT} done"
done
