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

log_info "Creating Sqrt variants (base, newton-2, newton-1, Light)"

# Exact — uses math.sqrt (hardware FPU)
$SERVERLEDGE create \
  --function Sqrt-base \
  --logical-name Sqrt \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/Sqrt.py \
  --runtime python310 \
  --handler Sqrt.handler \
  --variant-id base

# Approximate (2 Newton iterations) — error ≈ 0.1%
$SERVERLEDGE create \
  --function Sqrt-newton2 \
  --logical-name Sqrt \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/Sqrt_newton2.py \
  --runtime python310 \
  --handler Sqrt_newton2.handler \
  --variant-id newton-2

# Approximate (1 Newton iteration) — error ≈ 2%
$SERVERLEDGE create \
  --function Sqrt-newton1 \
  --logical-name Sqrt \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/Sqrt_newton1.py \
  --runtime python310 \
  --handler Sqrt_newton1.handler \
  --variant-id newton-1

# Approximate (bit-shift only) — error ≈ 4.3%, cheapest
$SERVERLEDGE create \
  --function Sqrt-light \
  --logical-name Sqrt \
  --memory 256 \
  --src ../../examples/Tesi/Sqrt/SqrtLight.py \
  --runtime python310 \
  --handler SqrtLight.handler \
  --variant-id Light

log_ok "All Sqrt variants created"
