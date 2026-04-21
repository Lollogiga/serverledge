#!/bin/bash
set -euo pipefail

# =====================================================
# SchedulerMonteCarlo.sh — Variant-selection per MonteCarlo
#
# Varianti (fronte di Pareto energy vs error):
#
#  Variant   N         error_estimate  energy/inv   atteso CI
#  n1000     1 000     0.052           ~0.000147 J  PL  ~750 gCO2/kWh
#  n5000     5 000     0.023           ~0.000912 J  DE  ~350 gCO2/kWh
#  n10000    10 000    0.016           ~0.002023 J  IT-NO ~200
#  n50000    50 000    0.0074          ~0.014308 J  FR   ~60
#  n100000   100 000   0.0052          ~0.033710 J  NO-NO5 ~20
#  n500000   500 000   0.0023          ~0.231480 J  rete molto pulita
#
# Tutte le varianti sono Pareto-ottimali: al crescere di N,
# l'energia cresce ma l'errore scende (nessuna variante domina).
# Logica analoga a PiLeibniz: CI alta → n1000, CI bassa → n500000.
# =====================================================

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { echo -e "\e[31m[ERROR]\e[0m serverledge-cli not found"; exit 1; }

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log()  { echo -e "${BLUE}[INFO]${RESET} $1"; }
ok()   { echo -e "${GREEN}[OK]${RESET}   $1"; }
warn() { echo -e "${YELLOW}[WARN]${RESET} $1"; }
sep()  { echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"; }

# =====================================================
# CLEANUP
# =====================================================
log "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      log "  Deleting: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done
ok "Cleanup completed"

# =====================================================
# CREATE — funzione base con --approximate
# Carica automaticamente variants/MonteCarlo/MonteCarlo.json
# registrando le 6 varianti: n1000, n5000, n10000, n50000, n100000, n500000
# =====================================================
log "Creating MonteCarlo base function (--approximate)"

$SERVERLEDGE create \
  --function MonteCarlo \
  --runtime python310 \
  --src ../../variants/MonteCarlo/montecarlo_n1000.py \
  --handler montecarlo_n1000.handler \
  --memory 128 \
  --approximate

ok "MonteCarlo created (variants loaded automatically)"
sleep 3

# =====================================================
# INVOKE 1 — legacy (nessuna variant-selection)
# Atteso: esegue la variante base (n1000)
# =====================================================
sep
log "INVOKE 1 — legacy (no allowApprox)  →  atteso: n1000"
$SERVERLEDGE invoke --function MonteCarlo
ok "Invoke 1 completato"
sleep 2

# =====================================================
# INVOKE 2 — quality-weight=0.0 → min energia
# Atteso: n1000  (energia minima, errore più alto)
# Nota: non supportato come flag CLI → uso --ci-zone PL per forzare la zona più sporca
# =====================================================
sep
log "INVOKE 2 — ci-zone=PL  (λ basso, rete sporca)  →  atteso: n1000"
$SERVERLEDGE invoke \
  --function MonteCarlo \
  --ci-zone PL
ok "Invoke 2 completato"
sleep 2

# =====================================================
# INVOKE 3 — trade-off: ci-zone IT-NO (~200 gCO2/kWh)
# Atteso: n10000 / n50000
# =====================================================
sep
log "INVOKE 3 — ci-zone=IT-NO  (λ~0.63)  →  atteso: n10000 / n50000"
$SERVERLEDGE invoke \
  --function MonteCarlo \
  --ci-zone IT-NO
ok "Invoke 3 completato"
sleep 2

# =====================================================
# INVOKE 4 — min errore: ci-zone NO-NO5 (~20 gCO2/kWh)
# Atteso: n500000  (errore minimo, energia massima)
# =====================================================
sep
log "INVOKE 4 — ci-zone=NO-NO5  (λ alto, rete pulita)  →  atteso: n500000"
$SERVERLEDGE invoke \
  --function MonteCarlo \
  --ci-zone NO-NO5
ok "Invoke 4 completato"
sleep 2

# =====================================================
# INVOKE 5–9 — per zona CI
#
# CI approssimativa in gCO2/kWh:
#   NO-NO5 ~20   idroelettrica → λ alto → n100000 / n500000
#   FR     ~60   nucleare      → λ~0.71 → n50000
#   IT-NO  ~200  mix           → λ~0.63 → n10000 / n50000
#   DE     ~350  mix/carbone   → λ~0.55 → n5000  / n10000
#   PL     ~750  carbone       → λ~0.33 → n1000
# =====================================================
invoke_ci_zone() {
    local zone="$1"
    sep
    if [[ -n "$zone" ]]; then
        log "INVOKE ci-zone=${zone}"
        $SERVERLEDGE invoke --function MonteCarlo --ci-zone "$zone"
    else
        log "INVOKE default zone (da serverledge-conf.yaml)"
        $SERVERLEDGE invoke --function MonteCarlo
    fi
    ok "Invoke ci-zone=${zone:-default} completato"
    sleep 1
}

invoke_ci_zone "FR"     # ~60  gCO2/kWh → n50000
invoke_ci_zone "DE"     # ~350 gCO2/kWh → n5000/n10000

sep
ok "SchedulerMonteCarlo completato."
