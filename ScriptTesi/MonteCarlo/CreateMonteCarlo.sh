#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

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

# -------------------------------------------------------
# MonteCarlo Pi — 6 varianti con N crescente
#
# Tutte approssimate. L'errore decresce come 1/√N (convergenza
# stocastica), a differenza di PiLeibniz che scala come 1/N.
#
# La σ teorica è: 4·√(p(1−p)/N)  con p=π/4≈0.7854
#
#  Variant    N          error_estimate (1σ)   Energy/inv (stima)
#  n1000      1 000      0.052                 ~0.000214 J
#  n5000      5 000      0.023                 ~0.000314 J
#  n10000     10 000     0.016                 ~0.000439 J
#  n50000     50 000     0.0074                ~0.001439 J
#  n100000    100 000    0.0052                ~0.002689 J
#  n500000    500 000    0.0023                ~0.012689 J
#
# NOTE: invocation_joule è una stima iniziale.
#       Esegui ProfileMonteCarlo.sh per sostituire con valori misurati.
# -------------------------------------------------------

for N in 1000 5000 10000 50000 100000 500000; do
  log_info "Creating montecarlo-n${N}  (N=${N})"
  $SERVERLEDGE create \
    --function "montecarlo-n${N}" \
    --runtime python310 \
    --src "../../examples/Tesi/MonteCarlo/montecarlo_n${N}.py" \
    --handler "montecarlo_n${N}.handler" \
    --memory 128 \
  log_ok "montecarlo-n${N} created"
done

log_ok "All MonteCarlo variants created"
log_ok "Workflow completed successfully"
