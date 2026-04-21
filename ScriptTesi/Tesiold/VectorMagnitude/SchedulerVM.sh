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
# INVOKE 1: default zone (da serverledge-conf.yaml)
# -----------------------
log "Invoking VectorMagnitude (default zone)"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y}
ok "Default invocation completed"

# -----------------------
# INVOKE 2: CI zona pulita NO-NO5 (~20 gCO2/kWh)
# λ alto → base (error=0)
# -----------------------
log "Invoking VectorMagnitude ci-zone=NO-NO5 (rete pulita, atteso: base)"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y} \
  --ci-zone NO-NO5
ok "NO-NO5 invocation completed"

# -----------------------
# INVOKE 3: CI zona media IT-NO (~200 gCO2/kWh)
# λ~0.63 → Light
# -----------------------
log "Invoking VectorMagnitude ci-zone=IT-NO (mix, atteso: Light)"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y} \
  --ci-zone IT-NO
ok "IT-NO invocation completed"

# -----------------------
# INVOKE 4: CI zona sporca PL (~750 gCO2/kWh)
# λ basso → chebyshev (cheapest)
# -----------------------
log "Invoking VectorMagnitude ci-zone=PL (rete sporca, atteso: chebyshev)"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y} \
  --ci-zone PL
ok "PL invocation completed"

ok "SchedulingVector completed"
