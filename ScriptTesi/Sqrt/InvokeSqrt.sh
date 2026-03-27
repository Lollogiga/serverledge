#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"

VALUE=987654321.123

for VARIANT in Sqrt-base Sqrt-newton2 Sqrt-newton1 Sqrt-light; do
  log_info "Invoking ${VARIANT} with n=${VALUE}"
  $SERVERLEDGE invoke \
    --function "${VARIANT}" \
    --param n:${VALUE}
  log_ok "${VARIANT} done"
done


