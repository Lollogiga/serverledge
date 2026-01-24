#!/bin/bash
set -euo pipefail

# =====================================================
# COLORI PER I LOG
# =====================================================
BLUE="\e[34m"
GREEN="\e[32m"
YELLOW="\e[33m"
RED="\e[31m"
RESET="\e[0m"

log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

# =====================================================
# PERCORSO CLI SERVERLEDGE
# =====================================================
SERVERLEDGE="../../bin/serverledge-cli"

if [[ ! -x "$SERVERLEDGE" ]]; then
  log_error "serverledge-cli not found or not executable"
  exit 1
fi

# =====================================================
# CLEANUP FUNZIONI SERVERLEDGE
# =====================================================
log_info "Cleaning existing Serverledge functions"

$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      log_info "Deleting function: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done

log_ok "Serverledge functions cleanup completed"

# =====================================================
# CREATE – SOLO FUNZIONE BASE (VARIANTI AUTO)
# =====================================================
log_info "Creating Fibonacci base function (approximate enabled)"

$SERVERLEDGE create \
  --function fibonacci \
  --memory 128 \
  --src ../../examples/Tesi/fibonacci/fibonacci.py \
  --runtime python310 \
  --handler fibonacci.handler \
  --input "n:Int" \
  --output "y:Int" \
  --approximate

log_ok "Fibonacci base function created (variants loaded automatically)"

sleep 3

# =====================================================
# INVOKE 1 – LEGACY (NO APPROX)
# =====================================================
log_info "Invoking Fibonacci (legacy, base only)"

$SERVERLEDGE invoke \
  --function fibonacci \
  --param n:10

log_ok "Legacy invocation completed"

sleep 2

# =====================================================
# INVOKE 2 – ALLOW APPROX
# =====================================================
log_info "Invoking Fibonacci with allowApprox"

$SERVERLEDGE invoke \
  --function fibonacci \
  --allowApprox \
  --param n:10

log_ok "allowApprox invocation completed"

sleep 2

# =====================================================
# INVOKE 3 – ALLOW APPROX + BUDGET
# =====================================================
log_info "Invoking Fibonacci with allowApprox + energy budget"

$SERVERLEDGE invoke \
  --function fibonacci \
  --allowApprox \
  --maxEnergyJoule 0.0000000001 \
  --param n:10

log_ok "Energy-constrained invocation completed"

# =====================================================
# DONE
# =====================================================
log_ok "Scheduler Fibonacci workflow completed successfully"
