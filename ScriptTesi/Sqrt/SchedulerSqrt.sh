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
log "Creating Sqrt base function (approximate enabled)"
# ⚠️ Adatta src/handler ai tuoi esempi reali se differiscono
$SERVERLEDGE create \
  --function Sqrt \
  --runtime python310 \
  --src ../../examples/Tesi/Sqrt/Sqrt.py \
  --handler sqrt.handler \
  --memory 128 \
  --input "value:Float" \
  --output "res:Float" \
  --approximate

ok "Sqrt created (variants loaded automatically)"

# -----------------------
# INVOKE 1: legacy
# -----------------------
log "Invoking Sqrt (legacy)"
$SERVERLEDGE invoke --function Sqrt --param n:12345.678
ok "Legacy invocation completed"

# -----------------------
# INVOKE 2: allowApprox
# -----------------------
log "Invoking Sqrt with allowApprox"
$SERVERLEDGE invoke --function Sqrt --allowApprox --param n:12345.678
ok "allowApprox invocation completed"

# -----------------------
# INVOKE 3: strict budget (expected reject)
# -----------------------
log "Invoking Sqrt with allowApprox + strict energy budget (expected reject)"
if ! $SERVERLEDGE invoke --function Sqrt --allowApprox --maxEnergyJoule 0.0000000001 --param n:12345.678; then
  warn "Expected: rejected by energy policy"
fi

ok "SchedulingSqrt completed"
