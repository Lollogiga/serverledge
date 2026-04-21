#!/bin/bash
set -euo pipefail

SERVERLEDGE="../../bin/serverledge-cli"

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log() { echo -e "${BLUE}[INFO]${RESET} $1"; }
ok()  { echo -e "${GREEN}[OK]${RESET}   $1"; }
warn(){ echo -e "${YELLOW}[WARN]${RESET} $1"; }

# -----------------------
# CLEANUP
# -----------------------
log "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null | tr -d '[]",' | sed 's/^[[:space:]]*//' | sed '/^$/d' | while read -r fn; do
  log "Deleting function: $fn"
  $SERVERLEDGE delete --function "$fn" || true
done
ok "Cleanup completed"

# -----------------------
# CREATE
# -----------------------
log "Creating SentimentAnalysis base function (approximate enabled)"
$SERVERLEDGE create \
  --function SentimentAnalysis \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SAHeavy.py \
  --handler SAHeavy.handler \
  --memory 2048 \
  --input "text:Text" \
  --output "label:Text, confidence:Float" \
  --approximate

ok "SentimentAnalysis created (variants loaded automatically)"

# -----------------------
# Helper ML-tolerant
# -----------------------
sa_invoke() {
  local label="$1"; shift
  log "$label"
  if $SERVERLEDGE invoke "$@"; then
    ok "Completato: $label"
  else
    rc=$?
    if [[ $rc -eq 2 ]]; then
      warn "HTTP 500 — modello ML non pre-scaricato nel container (atteso in ambienti senza modelli)"
    else
      log "Invocation exit=$rc (label: $label)"
      exit $rc
    fi
  fi
}

# -----------------------
# INVOKE 1: default zone (da serverledge-conf.yaml)
# -----------------------
sa_invoke "default zone" \
  --function SentimentAnalysis --param text:"Looks good but it works terribly"

# -----------------------
# INVOKE 2: CI zona pulita NO-NO5 (~20 gCO2/kWh)
# λ alto → roberta-large (max quality)
# -----------------------
sa_invoke "ci-zone=NO-NO5 (rete pulita, atteso: roberta-large)" \
  --function SentimentAnalysis --param text:"Looks good but it works terribly" --ci-zone NO-NO5

# -----------------------
# INVOKE 3: CI zona media DE (~350 gCO2/kWh)
# λ~0.55 → distilbert
# -----------------------
sa_invoke "ci-zone=DE (mix, atteso: distilbert)" \
  --function SentimentAnalysis --param text:"Looks good but it works terribly" --ci-zone DE

# -----------------------
# INVOKE 4: CI zona sporca PL (~750 gCO2/kWh)
# λ basso → vader (cheapest)
# -----------------------
sa_invoke "ci-zone=PL (rete sporca, atteso: vader)" \
  --function SentimentAnalysis --param text:"Looks good but it works terribly" --ci-zone PL

ok "SchedulingSA completed"
