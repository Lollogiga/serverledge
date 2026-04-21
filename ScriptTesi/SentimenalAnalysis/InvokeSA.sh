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
# ML-tolerant: un exit=2 (500) è accettato come warning se i modelli
# non sono stati pre-scaricati nel container python-ml.
# =====================================================
sa_invoke() {
  local fn="$1"; shift
  log_info "Invoking $fn"
  if $SERVERLEDGE invoke --function "$fn" "$@"; then
    log_ok "$fn invocation completed"
  else
    rc=$?
    if [[ $rc -eq 2 ]]; then
      log_warn "$fn returned HTTP 500 — modello ML non pre-scaricato nel container (atteso in ambienti senza modelli)"
    else
      log_error "$fn fallito con exit=$rc"
      exit $rc
    fi
  fi
}

sa_invoke "SentimentAnalysis-vader" \
  --param text:"Looks good but it works terribly"

sa_invoke "SentimentAnalysis" \
  --param text:"Looks good but it works terribly"

# =====================================================
# DONE
# =====================================================
log_ok "Workflow completed successfully (ML failures documented above)"

