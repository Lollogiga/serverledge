#!/bin/bash
set -euo pipefail

SERVERLEDGE="../../bin/serverledge-cli"

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
ok()   { echo -e "${GREEN}[OK]${RESET}    $1"; }
warn() { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
sep()  { echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"; }

# ─────────────────────────────────────────────────────────────
# CLEANUP
# ─────────────────────────────────────────────────────────────
log "Cleaning existing Serverledge functions"
$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' | sed 's/^[[:space:]]*//' | sed '/^$/d' \
  | while read -r fn; do
      log "Deleting function: $fn"
      $SERVERLEDGE delete --function "$fn" || true
    done
ok "Cleanup completed"

# ─────────────────────────────────────────────────────────────
# CREATE – base con varianti (--approximate carica PiLeibniz.json)
# ─────────────────────────────────────────────────────────────
log "Creating PiLeibniz base function (--approximate)"
$SERVERLEDGE create \
  --function PiLeibniz \
  --runtime python310 \
  --src ../../variants/PiLeibniz/piLeibniz.py \
  --handler piLeibniz.handler \
  --memory 128 \
  --input "n:Int" \
  --output "y:Float" \
  --approximate
ok "PiLeibniz created — 9 varianti caricate da variants/PiLeibniz/"

sleep 2

# ─────────────────────────────────────────────────────────────
# Varianti nel JSON (9 totali):
#   Pareto-ottimali attese:
#     n1000   E=0.000200  Err=0.001000
#     n5000   E=0.000300  Err=0.000200
#     n10000  E=0.000430  Err=0.000100
#     n50000  E=0.000800  Err=0.000015
#     c       E=0.001037  Err=0.000000
#   Dominate (escluse dall'algoritmo):
#     n100000   E=0.001150  Err=0.000008  ← dominata da c
#     n200000   E=0.001550  Err=0.000003  ← dominata da c
#     base      E=0.007750  Err=0.000000  ← dominata da c
#     opt-py    E=0.007400  Err=0.000000  ← dominata da c
# ─────────────────────────────────────────────────────────────

# ─────────────────────────────────────────────────────────────
# INVOKE 1 – λ=0.0  →  minimizza energia  →  atteso: n1000
# ─────────────────────────────────────────────────────────────
sep
log "INVOKE 1 — quality-weight=0.0  (min energia — atteso: n1000)"
$SERVERLEDGE invoke \
  --function PiLeibniz \
  --quality-weight 0.0
ok "Invoke 1 completato"

sleep 2

# ─────────────────────────────────────────────────────────────
# INVOKE 2 – λ=1.0  →  minimizza errore  →  atteso: c (Err=0)
# ─────────────────────────────────────────────────────────────
sep
log "INVOKE 2 — quality-weight=1.0  (min errore — atteso: c)"
$SERVERLEDGE invoke \
  --function PiLeibniz \
  --quality-weight 1.0
ok "Invoke 2 completato"

sleep 2

# ─────────────────────────────────────────────────────────────
# INVOKE 3 – λ=0.5  →  trade-off bilanciato  →  atteso: n5000
# E_range=0.000837  Err_range=0.001000
# score(n1000) =0.5*0.000 +0.5*1.000=0.500
# score(n5000) =0.5*0.120 +0.5*0.200=0.160  ← MIN
# score(n10000)=0.5*0.275 +0.5*0.100=0.187
# score(n50000)=0.5*0.717 +0.5*0.015=0.366
# score(c)     =0.5*1.000 +0.5*0.000=0.500
# ─────────────────────────────────────────────────────────────
sep
log "INVOKE 3 — quality-weight=0.5  (trade-off bilanciato — atteso: n5000)"
$SERVERLEDGE invoke \
  --function PiLeibniz \
  --quality-weight 0.5
ok "Invoke 3 completato"

sleep 2

# ─────────────────────────────────────────────────────────────
# INVOKE 4 – λ=0.2  →  orientato all'energia  →  atteso: n5000
# score(n1000) =0.8*0.000 +0.2*1.000=0.200
# score(n5000) =0.8*0.120 +0.2*0.200=0.136  ← MIN
# score(n10000)=0.8*0.275 +0.2*0.100=0.240
# score(n50000)=0.8*0.717 +0.2*0.015=0.577
# score(c)     =0.8*1.000 +0.2*0.000=0.800
# ─────────────────────────────────────────────────────────────
sep
log "INVOKE 4 — quality-weight=0.2  (lieve peso errore — atteso: n5000)"
$SERVERLEDGE invoke \
  --function PiLeibniz \
  --quality-weight 0.2
ok "Invoke 4 completato"

sep
ok "SchedulingPiLiebniz completato."
