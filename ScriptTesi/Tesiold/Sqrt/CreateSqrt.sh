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

# Creates Sqrt (base) + Sqrt-newton-2 + Sqrt-newton-1 + Sqrt-Light automatically
log_info "Creating Sqrt with all variants (--approximate)"
$SERVERLEDGE create \
  --function Sqrt \
  --runtime python310 \
  --src ../../variants/Sqrt/Sqrt.py \
  --handler Sqrt.handler \
  --memory 256 \
  --approximate

log_ok "All Sqrt variants created (Sqrt / Sqrt-newton-2 / Sqrt-newton-1 / Sqrt-Light)"
