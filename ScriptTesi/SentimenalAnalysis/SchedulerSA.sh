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
# INVOKE 1: legacy
# -----------------------
log "Invoking SentimentAnalysis (legacy)"
$SERVERLEDGE invoke --function SentimentAnalysis --param text:"Looks good but it works terribly"
ok "Legacy invocation completed"


# -----------------------
# INVOKE 2: allowApprox
# -----------------------
log "Invoking SentimentAnalysis with allowApprox"
$SERVERLEDGE invoke --function SentimentAnalysis --allowApprox --param text::"Looks good but it works terribly"
ok "allowApprox invocation completed"

# -----------------------
# INVOKE 3: strict budget (expected reject)
# -----------------------
log "Invoking SentimentAnalysis with allowApprox + strict energy budget (expected reject)"
if ! $SERVERLEDGE invoke --function SentimentAnalysis --allowApprox --maxEnergyJoule 0.0000000001 --param text:"Looks good but it works terribly"; then
  warn "Expected: rejected by energy policy"
fi

ok "SchedulingSA completed"
