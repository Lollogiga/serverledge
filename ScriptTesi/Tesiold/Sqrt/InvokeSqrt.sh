#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; RESET="\e[0m"
log_info(){ echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok(){ echo -e "${GREEN}[OK]${RESET}    $1"; }

SERVERLEDGE="../../bin/serverledge-cli"

VALUE=987654321.123

for VARIANT in Sqrt Sqrt-newton-2 Sqrt-newton-1 Sqrt-Light; do
  log_info "Invoking ${VARIANT} with n=${VALUE}"
  $SERVERLEDGE invoke \
    --function "${VARIANT}" \
    --param n:${VALUE}
  log_ok "${VARIANT} done"
done


