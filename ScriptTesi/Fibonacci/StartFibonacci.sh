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
# =====================================================
# CLEANUP FUNZIONI SERVERLEDGE (JSON multiline)
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
# CREAZIONE FUNZIONE FIBONACCI (C / native)
# =====================================================
log_info "Creating Fibonacci function (C, native runtime)"

$SERVERLEDGE create \
  --function fibonacci-c \
  --memory 128 \
  --src ../../examples/Tesi/fibonacci/fibonacci \
  --runtime native \
  --handler fibonacci \
  --input "x:Int" \
  --output "y:Int"

log_ok "Function fibonacci-c created"

# =====================================================
# CREAZIONE FUNZIONE FIBONACCI (python)
# =====================================================
$SERVERLEDGE create \
  --function fibonacci-py \
  --memory 128 \
  --src ../../examples/Tesi/fibonacci/fibonacci.py \
  --runtime python310 \
  --handler fibonacci.handler \
  --input "x:Int" \
  --output "y:Int"

log_ok "Function fibonacci-py created"

# =====================================================
# CREAZIONE FUNZIONE FIBONACCI BINET (python)
# =====================================================
$SERVERLEDGE create \
  --function fibonacci_o-py \
  --memory 128 \
  --src ../../examples/Tesi/fibonacci/fibonacci_o.py \
  --runtime python310 \
  --handler fibonacci_o.handler \
  --input "x:Int" \
  --output "y:Int"

log_ok "Function fibonacci_o created"

# =====================================================
# DONE
# =====================================================
log_ok "Workflow completed successfully"
