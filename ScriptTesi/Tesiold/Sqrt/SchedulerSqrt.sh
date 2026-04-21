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
# INVOKE 1: default zone (da serverledge-conf.yaml)
# -----------------------
log "Invoking Sqrt (default zone)"
$SERVERLEDGE invoke --function Sqrt --param n:12345.678
ok "Default invocation completed"

# -----------------------
# INVOKE 2: CI zona pulita (NO-NO5 ~20 gCO2/kWh)
# λ alto → priorità qualità → base (error=0)
# -----------------------
log "Invoking Sqrt ci-zone=NO-NO5 (rete pulita, atteso: base)"
$SERVERLEDGE invoke --function Sqrt --param n:12345.678 --ci-zone NO-NO5
ok "NO-NO5 invocation completed"

# -----------------------
# INVOKE 3: CI zona media (IT-NO ~200 gCO2/kWh)
# λ~0.63 → trade-off → newton-2 / newton-1
# -----------------------
log "Invoking Sqrt ci-zone=IT-NO (mix, atteso: newton-2 / newton-1)"
$SERVERLEDGE invoke --function Sqrt --param n:12345.678 --ci-zone IT-NO
ok "IT-NO invocation completed"

# -----------------------
# INVOKE 4: CI zona sporca (PL ~750 gCO2/kWh)
# λ basso → priorità energia → Light (cheapest)
# -----------------------
log "Invoking Sqrt ci-zone=PL (rete sporca, atteso: Light)"
$SERVERLEDGE invoke --function Sqrt --param n:12345.678 --ci-zone PL
ok "PL invocation completed"

ok "SchedulingSqrt completed"
