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

log_info "Creating VectorMagnitude variants (base, Light, chebyshev)"

# Exact — math.hypot
$SERVERLEDGE create \
  --function VectorMagnitude-base \
  --logical-name VectorMagnitude \
  --memory 256 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitude.py \
  --runtime python310 \
  --handler VectorMagnitude.handler \
  --variant-id base

# Approximate (alpha-max-plus-beta) — error ≤ 4%
$SERVERLEDGE create \
  --function VectorMagnitude-light \
  --logical-name VectorMagnitude \
  --memory 256 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitudeLight.py \
  --runtime python310 \
  --handler VectorMagnitudeLight.handler \
  --variant-id Light

# Approximate (Chebyshev / max component) — error ≤ 29%
$SERVERLEDGE create \
  --function VectorMagnitude-chebyshev \
  --logical-name VectorMagnitude \
  --memory 256 \
  --src ../../examples/Tesi/VectorMagnitude/VectorMagnitudeChebyshev.py \
  --runtime python310 \
  --handler VectorMagnitudeChebyshev.handler \
  --variant-id chebyshev

log_ok "All VectorMagnitude variants created"