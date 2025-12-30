#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"

# Parametri di test
VAL_X=3310.12
VAL_Y=443212.1

# =====================================================
# SINGLE INVOKE LIGHT
# =====================================================

log_info "Invoking VectorMagnitudeLight-py (x=${VAL_X}, y=${VAL_Y})"
$SERVERLEDGE invoke \
  --function VectorMagnitudeLight-py \
  --param x:${VAL_X} \
  --param y:${VAL_Y}

echo "" # Spaziatura output
log_ok "Light version executed"

# =====================================================
# SINGLE INVOKE BASE
# =====================================================

log_info "Invoking VectorMagnitude-py (x=${VAL_X}, y=${VAL_Y})"
$SERVERLEDGE invoke \
  --function VectorMagnitude-py \
  --param x:${VAL_X} \
  --param y:${VAL_Y}

echo "" # Spaziatura output
log_ok "Base version executed"

sleep 2