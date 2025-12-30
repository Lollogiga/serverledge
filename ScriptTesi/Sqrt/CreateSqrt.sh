#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }
log_error(){ echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

log_info "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null | tr -d '[]",' | sed '/^$/d' | while read -r fn; do
  log_info "Deleting function: $fn"
  $SERVERLEDGE delete --function "$fn" || true
done
log_ok "Cleanup completed"

log_info "Creating Sqrt (Python / Python Light)"

$SERVERLEDGE create \
  --function Sqrt-py \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/Sqrt.py \
  --runtime python310 \
  --handler Sqrt.handler \
  --input "value:Float" \
  --output "res:Float"

$SERVERLEDGE create \
  --function SqrtLight-py \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/SqrtLight.py \
  --runtime python310 \
  --handler SqrtLight.handler \
  --input "value:Float" \
  --output "res:Float"

log_ok "Workflow completed successfully"
