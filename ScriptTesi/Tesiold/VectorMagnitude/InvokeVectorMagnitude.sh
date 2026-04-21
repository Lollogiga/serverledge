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
# INVOKE LIGHT (alpha-max-plus-beta, error ≤ 4%)
# =====================================================

log_info "Invoking VectorMagnitude-Light (x=${VAL_X}, y=${VAL_Y})"
$SERVERLEDGE invoke \
  --function VectorMagnitude-Light \
  --param x:${VAL_X} \
  --param y:${VAL_Y}

echo "" # Spaziatura output
log_ok "Light version executed"

# =====================================================
# INVOKE BASE (math.hypot)
# =====================================================

log_info "Invoking VectorMagnitude (base, x=${VAL_X}, y=${VAL_Y})"
$SERVERLEDGE invoke \
  --function VectorMagnitude \
  --param x:${VAL_X} \
  --param y:${VAL_Y}

echo "" # Spaziatura output
log_ok "Base version executed"