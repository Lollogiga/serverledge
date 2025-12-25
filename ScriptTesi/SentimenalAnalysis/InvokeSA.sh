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
# INVOKE (ESECUZIONE)
# =====================================================
log_info "Invoking SALight"

$SERVERLEDGE invoke \
  --function SALight \
  --param text:"I really like this product"

log_ok "Invocation completed"

log_info "Invoking SAMedium"

$SERVERLEDGE invoke \
  --function SAMedium \
  --param text:"I really like this product"

log_ok "Invocation completed"

log_info "Invoking SAHeavy-py"

$SERVERLEDGE invoke \
  --function SAHeavy \
  --param text:"I really like this product"

log_ok "Invocation completed"

# =====================================================
# DONE
# =====================================================
log_ok "Workflow completed successfully"

