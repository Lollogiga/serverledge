#!/bin/bash
set -euo pipefail

SERVERLEDGE="../../bin/serverledge-cli"

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log() { echo -e "${BLUE}[INFO]${RESET} $1"; }
ok()  { echo -e "${GREEN}[OK]${RESET}   $1"; }
warn(){ echo -e "${YELLOW}[WARN]${RESET} $1"; }

# -----------------------
# CLEANUP (all functions)
# -----------------------
log "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null | tr -d '[]",' | sed 's/^[[:space:]]*//' | sed '/^$/d' | while read -r fn; do
  log "Deleting function: $fn"
  $SERVERLEDGE delete --function "$fn" || true
done
ok "Cleanup completed"

# -----------------------
# CREATE (base only)
# -----------------------
log "Creating PiLeibniz base function (approximate enabled)"
# ⚠️ Adatta src/handler ai tuoi esempi reali se differiscono
$SERVERLEDGE create \
  --function PiLeibniz \
  --runtime python310 \
  --src ../../examples/Tesi/PiLeibnizFunction/piLeibniz.py \
  --handler piLeibniz.handler \
  --memory 128 \
  --input "n:Int" \
  --output "y:Float" \
  --approximate

ok "PiLeibniz created (variants loaded automatically)"

# -----------------------
# INVOKE 1: legacy
# -----------------------
log "Invoking PiLeibniz (legacy)"
$SERVERLEDGE invoke --function PiLeibniz --param n:200000
ok "Legacy invocation completed"

# -----------------------
# INVOKE 2: allowApprox
# -----------------------
log "Invoking PiLeibniz with allowApprox"
$SERVERLEDGE invoke --function PiLeibniz --allowApprox --param n:200000
ok "allowApprox invocation completed"

# -----------------------
# INVOKE 3: allowApprox + too strict budget (expected reject)
# -----------------------
log "Invoking PiLeibniz with allowApprox + strict energy budget (expected reject)"
if ! $SERVERLEDGE invoke --function PiLeibniz --allowApprox --maxEnergyJoule 0.0000000001 --param n:200000; then
  warn "Expected: rejected by energy policy"
fi

ok "SchedulingPiLeibniz completed"
