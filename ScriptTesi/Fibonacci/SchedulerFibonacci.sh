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
sep()       { echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"; }

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
# CREATE – FUNZIONE BASE CON VARIANTI (--approximate)
# =====================================================
# Il flag --approximate carica automaticamente fibonacci.json
# dalla directory variants/fibonacci/, registrando le 3 varianti:
#   base          E=0.000732  Err=0.0  (Python base)
#   optimization-py E=0.000495  Err=0.0  (Python ottimizzato)
#   c             E=0.000172  Err=0.0  (nativo C)
#
# Poiché tutte le varianti hanno Err=0.0, l'unica Pareto-ottimale
# è "c" che domina le altre per energia inferiore.
# Il fronte di Pareto è composto da un SINGOLO punto: caso degenere.
# =====================================================
log_info "Creating Fibonacci base function (--approximate)"

$SERVERLEDGE create \
  --function fibonacci \
  --memory 128 \
  --src ../../variants/fibonacci/fibonacci.py \
  --runtime python310 \
  --handler fibonacci.handler \
  --input "n:Int" \
  --output "y:Int" \
  --approximate

log_ok "Fibonacci base function created (variants loaded automatically)"

sleep 3

# =====================================================
# INVOKE 1 – λ=0.0  →  min energia  →  atteso: c
# Fronte di Pareto = {c} (punto singolo)
# Qualunque λ porta allo stesso risultato.
# =====================================================
sep
log_info "INVOKE 1 — quality-weight=0.0  (min energia — atteso: c)"
$SERVERLEDGE invoke \
  --function fibonacci \
  --param n:10 \
  --quality-weight 0.0
log_ok "Invoke 1 completato"

sleep 2

# =====================================================
# INVOKE 2 – λ=0.5  →  trade-off  →  atteso: c
# Fronte = {c}: un solo punto, score=0 indipendentemente da λ
# =====================================================
sep
log_info "INVOKE 2 — quality-weight=0.5  (trade-off — atteso: c)"
$SERVERLEDGE invoke \
  --function fibonacci \
  --param n:10 \
  --quality-weight 0.5
log_ok "Invoke 2 completato"

sleep 2

# =====================================================
# INVOKE 3 – λ=1.0  →  min errore  →  atteso: c
# Tutti hanno errore 0, c è comunque il Pareto-ottimale unico
# =====================================================
sep
log_info "INVOKE 3 — quality-weight=1.0  (min errore — atteso: c)"
$SERVERLEDGE invoke \
  --function fibonacci \
  --param n:10 \
  --quality-weight 1.0
log_ok "Invoke 3 completato"

# =====================================================
# DONE
# =====================================================
sep
log_ok "Scheduler Fibonacci workflow completato."
