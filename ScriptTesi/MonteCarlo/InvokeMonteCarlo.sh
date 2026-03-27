#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

for N in 1000 5000 10000 50000 100000 500000; do
  log_info "Invoking montecarlo-n${N}"
  $SERVERLEDGE invoke --function "montecarlo-n${N}"
  log_ok "montecarlo-n${N} done"
done

log_ok "All MonteCarlo invocations completed"
