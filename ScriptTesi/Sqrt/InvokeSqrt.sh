#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"

VALUE=987654321.123

# =====================================================
# INVOKE LIGHT
# =====================================================

log_info "Invoking SqrtLight-py"
$SERVERLEDGE invoke \
  --function SqrtLight-py \
  --param n:${VALUE}
log_ok "SqrtLight-py done"

# =====================================================
# INVOKE BASE
# =====================================================

log_info "Invoking Sqrt-py"
$SERVERLEDGE invoke \
  --function Sqrt-py \
  --param n:${VALUE}
log_ok "Sqrt-py done"


