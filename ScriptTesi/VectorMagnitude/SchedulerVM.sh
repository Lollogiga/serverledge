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
log "Creating VectorMagnitude base function (approximate enabled)"
# ⚠️ Adatta src/handler ai tuoi esempi reali se differiscono
$SERVERLEDGE create \
  --function VectorMagnitude \
  --runtime python310 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitude.py \
  --handler VectorMagnitude.handler \
  --memory 128 \
  --input "x:Float, y:Float" \
  --output "magnitude:Float" \
  --approximate

ok "VectorMagnitude created (variants loaded automatically)"


VAL_X=3310.12
VAL_Y=443212.1

# -----------------------
# INVOKE 1: legacy
# -----------------------
log "Invoking VectorMagnitude (legacy)"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y}

ok "Legacy invocation completed"

# -----------------------
# INVOKE 2: allowApprox
# -----------------------
log "Invoking VectorMagnitude with allowApprox"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y} \
  --allowApprox
ok "allowApprox invocation completed"

# -----------------------
# INVOKE 3: strict budget (expected reject)
# -----------------------
log "Invoking VectorMagnitude with allowApprox + strict energy budget (expected reject)"
if ! $SERVERLEDGE invoke \
       --function VectorMagnitude \
       --param x:${VAL_X} \
       --param y:${VAL_Y} \
       --allowApprox \
       --maxEnergyJoule 0.00001; then warn "Expected: rejected by energy policy"
fi

ok "SchedulingVector completed"
