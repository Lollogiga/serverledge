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
log_info "Creating PiLeibniz function (C, native runtime)"

$SERVERLEDGE create \
  --function piLeibniz-c \
  --memory 128 \
  --src ../../examples/Tesi/PiLeibnizFunction/piLeibniz \
  --runtime native \
  --handler piLeibniz \
  --input "n:Int" \
  --output "y:Float"

log_ok "Function piLeibniz-c created"

# =====================================================
# CREAZIONE FUNZIONE FIBONACCI (python)
# =====================================================
$SERVERLEDGE create \
  --function piLeibniz-py \
  --memory 128 \
  --src ../../examples/Tesi/PiLeibnizFunction/piLeibniz.py \
  --runtime python310 \
  --handler piLeibniz.handler \
  --input "n:Int" \
  --output "y:Float"

log_ok "Function piLeibniz-py created"

# =====================================================
# CREAZIONE FUNZIONE FIBONACCI BINET (python)
# =====================================================
$SERVERLEDGE create \
  --function piLeibniz_o-py \
  --memory 128 \
  --src ../../examples/Tesi/PiLeibnizFunction/piLeibniz_o.py \
  --runtime python310 \
  --handler piLeibniz_o.handler \
  --input "n:Int" \
  --output "y:Float"

log_ok "Function piLeibniz_o created"

# =====================================================
# DONE
# =====================================================
log_ok "Workflow completed successfully"
