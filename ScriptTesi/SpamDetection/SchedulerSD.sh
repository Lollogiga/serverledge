#!/bin/bash
set -euo pipefail

# =====================================================
# SchedulerSD.sh — Variant-selection per SpamDetection
#
# Varianti (fronte di Pareto energy vs quality):
#
#  Variant      quality_score  energy/inv   atteso con λ
#  roberta-spam 0.95           ~0.085 J     λ=1.0  (max quality)
#  distilbert   0.80           ~0.035 J     λ~0.7
#  bert-tiny    0.65           ~0.010 J     λ~0.4
#  keyword      0.50           ~0.002 J     λ=0.0  (min energy)
#
# Tutte le varianti sono Pareto-ottimali perché nessuna
# domina le altre su entrambi gli assi.
# =====================================================

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { echo -e "\e[31m[ERROR]\e[0m serverledge-cli not found"; exit 1; }

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log()  { echo -e "${BLUE}[INFO]${RESET} $1"; }
ok()   { echo -e "${GREEN}[OK]${RESET}   $1"; }
warn() { echo -e "${YELLOW}[WARN]${RESET} $1"; }
sep()  { echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"; }

SPAM_TEXT="Congratulations! You have won a \$1,000 prize! Click here to claim your reward NOW! Limited time offer — act fast!"

# CLEANUP
log "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      log "  Deleting: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done
ok "Cleanup completed"

# CREATE
log "Creating SpamDetection base function (--approximate)"
$SERVERLEDGE create \
  --function SpamDetection \
  --runtime python310 \
  --src ../../variants/SpamDetection/SDKeyword.py \
  --handler SDKeyword.handler \
  --memory 256 \
  --approximate
ok "SpamDetection created (variants loaded automatically)"
sleep 3

PARAMS_FILE=$(mktemp /tmp/sd_sched_XXXXXX.json)
trap 'rm -f "$PARAMS_FILE"' EXIT
python3 -c "import json, sys; print(json.dumps({'text': sys.argv[1]}))" "$SPAM_TEXT" > "$PARAMS_FILE"

sd_invoke() {
  local label="$1"; shift
  sep
  log "$label"
  if $SERVERLEDGE invoke --function SpamDetection \
       --params_file "$PARAMS_FILE" "$@"; then
    ok "Invoke completato: $label"
  else
    rc=$?
    if [[ $rc -eq 2 ]]; then
      warn "HTTP 500 — modello ML non pre-scaricato nel container (atteso in ambiente senza cache)"
    else
      log "Invocation exit=$rc (label: $label)"
      exit $rc
    fi
  fi
  sleep 2
}

sd_invoke "INVOKE 1 — default zone"
sd_invoke "INVOKE 2 — ci-zone=NO-NO5  (rete pulita)   →  atteso: roberta-spam"  --ci-zone NO-NO5
sd_invoke "INVOKE 3 — ci-zone=FR      (nucleare)      →  atteso: distilbert"    --ci-zone FR
sd_invoke "INVOKE 4 — ci-zone=DE      (mix/carbone)   →  atteso: bert-tiny"     --ci-zone DE
sd_invoke "INVOKE 5 — ci-zone=PL      (rete sporca)   →  atteso: keyword"       --ci-zone PL
sd_invoke "INVOKE 6 — ci-zone=IT-NO   (λ~0.63)"                                 --ci-zone IT-NO

sep
ok "SchedulerSD completato."
