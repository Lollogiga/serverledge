#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"

VALUE=987654321.123
REPS=40

# =====================================================
# INVOKE LIGHT
# =====================================================

log_info "Invoking SqrtLight-py (${REPS} times, value=${VALUE})"
for ((i=1; i<=REPS; i++)); do
  $SERVERLEDGE invoke \
    --function SqrtLight-py \
    --param n:${VALUE} > /dev/null
done
log_ok "SqrtLight-py done"

# =====================================================
# INVOKE BASE
# =====================================================

log_info "Invoking Sqrt-py (${REPS} times, value=${VALUE})"
for ((i=1; i<=REPS; i++)); do
  $SERVERLEDGE invoke \
    --function Sqrt-py \
    --param n:${VALUE} > /dev/null
done
log_ok "Sqrt-py done"

sleep 2   # separazione visiva su Kepler


