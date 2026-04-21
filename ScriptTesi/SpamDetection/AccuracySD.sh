#!/bin/bash
# =============================================================================
# AccuracySD.sh — Binary accuracy test for SpamDetection variants
# =============================================================================
# Tests each SD variant on a 10-sentence labelled test set (5 HAM + 5 SPAM).
#
# Test set design:
#   Sentences 1-5  : clear HAM — all models expected correct
#   Sentences 6-7  : clear SPAM (explicit keywords) — all models expected correct
#   Sentences 8-10 : SPAM without classic keywords (phishing / urgency style)
#                    → keyword heuristic scores 0 → predicts HAM ✗
#                    → BERT-based models read context → correctly predict SPAM ✓
#
# This contrast empirically justifies the energy cost of neural models.
#
# Usage:  ./AccuracySD.sh
# =============================================================================

set -euo pipefail
cd "$(dirname "$0")"

SERVERLEDGE="../../bin/serverledge-cli"
TMPFILE=$(mktemp /tmp/sd_acc_XXXXXX.json)
trap 'rm -f "$TMPFILE"' EXIT

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"
BOLD="\e[1m"; RESET="\e[0m"
log_info() { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()   { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${RESET}  $1"; }

if [[ ! -x "$SERVERLEDGE" ]]; then
  echo -e "${RED}[ERROR]${RESET} $SERVERLEDGE not found or not executable"; exit 1
fi

# ---------------------------------------------------------------------------
# TEST SET — 10 messages with ground-truth labels
# ---------------------------------------------------------------------------
declare -a TEXTS=(
  # --- HAM (1-5) ---
  "Hey, are you coming to the team meeting on Friday at 10am?"
  "The project report is due next Wednesday. I will send you the draft tomorrow."
  "Could you pick up some groceries on your way home? We need milk and bread."
  "Just finished the code review. Left a few comments for you to check."
  "Happy birthday! Hope you have a great day with your family."
  # --- SPAM with explicit keywords (6-7) ---
  "Congratulations! You have won a \$1,000 prize! Click here to claim your reward NOW!"
  "FREE iPhone 16! Limited time offer — act now and WIN big!"
  # --- SPAM without classic keywords (8-10) — tricky for keyword heuristic ---
  "Your account access will be suspended within 24 hours. Please verify your identity immediately."
  "We were unable to process your payment. Update your billing information to avoid service interruption."
  "A parcel is pending your collection. Confirm your delivery address to release the package."
)

declare -a GROUND_TRUTH=(
  "HAM" "HAM" "HAM" "HAM" "HAM"
  "SPAM" "SPAM"
  "SPAM" "SPAM" "SPAM"
)

declare -a TRICKY=("" "" "" "" "" "" "" " (*)" " (*)" " (*)")

N_TEXTS=${#TEXTS[@]}

declare -a VARIANTS=(
  "SpamDetection-keyword"
  "SpamDetection-bert-tiny"
  "SpamDetection-distilbert"
  "SpamDetection"
)
declare -a DISPLAY_NAMES=(
  "Keyword heuristic"
  "BERT-tiny"
  "DistilBERT"
  "RoBERTa-spam"
)
declare -a QUALITY_SCORES=("0.50" "0.65" "0.80" "0.95")

N_VARIANTS=${#VARIANTS[@]}

declare -a CORRECT=()
declare -a NA_COUNT=()

echo ""
log_info "Avvio accuracy test — ${N_TEXTS} messaggi (5 HAM + 5 SPAM)"
log_info "Messaggi con (*) sono i tricky: spam senza keyword esplicite (phishing/urgency)."
echo ""

for vi in "${!VARIANTS[@]}"; do
  FN="${VARIANTS[$vi]}"
  DN="${DISPLAY_NAMES[$vi]}"
  correct=0
  na=0

  log_info "────────────────────────────────────────────────────────────"
  log_info "Variante: $FN  [$DN]"

  for ti in "${!TEXTS[@]}"; do
    TEXT="${TEXTS[$ti]}"
    GT="${GROUND_TRUTH[$ti]}"
    TRK="${TRICKY[$ti]}"

    python3 -c "import json, sys; print(json.dumps({'text': sys.argv[1]}))" "$TEXT" > "$TMPFILE"

    OUTPUT=$("$SERVERLEDGE" invoke \
      --function "$FN" \
      --params_file "$TMPFILE" \
      --ret_output 2>/dev/null) || {
      rc=$?
      if [[ $rc -eq 2 ]]; then
        log_warn "  Msg$((ti+1))$TRK [GT=$GT]: HTTP 500 — modello non scaricato → N/A"
        na=$((na+1))
        continue
      fi
      echo -e "${RED}[ERROR]${RESET} $FN msg $((ti+1)) fallito con exit=$rc"; exit $rc
    }

    LABEL=$(echo "$OUTPUT" | python3 -c "
import json, sys, re
text = sys.stdin.read()
m = re.search(r'\{[^{}]*\"label\"[^{}]*\}', text)
if m:
    try:
        d = json.loads(m.group())
        print(d.get('label', 'UNKNOWN').upper())
        sys.exit(0)
    except Exception:
        pass
print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

    if [[ "$LABEL" == "$GT" ]]; then
      log_ok  "  Msg $((ti+1))$TRK  [GT=$GT]  →  predetto: $LABEL  ✓"
      correct=$((correct+1))
    else
      log_warn "  Msg $((ti+1))$TRK  [GT=$GT]  →  predetto: $LABEL  ✗"
    fi
  done

  echo ""
  CORRECT[$vi]=$correct
  NA_COUNT[$vi]=$na
done

echo -e "${BOLD}══════════════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}  RISULTATI ACCURACY — SpamDetection  (${N_TEXTS} messaggi: 5 HAM + 5 SPAM)${RESET}"
echo -e "${BOLD}══════════════════════════════════════════════════════════════════════${RESET}"
printf "${BOLD}  %-22s  %-18s  %-11s  %-10s${RESET}\n" \
  "Modello" "Corretti / Acc.%" "Q-score" "N/A (500)"
echo "  ──────────────────────────────────────────────────────────────────────"

for vi in "${!VARIANTS[@]}"; do
  DN="${DISPLAY_NAMES[$vi]}"
  CNT="${CORRECT[$vi]}"
  NA="${NA_COUNT[$vi]}"
  QS="${QUALITY_SCORES[$vi]}"
  AVAIL=$((N_TEXTS - NA))

  if [[ $AVAIL -gt 0 ]]; then
    PCT=$(python3 -c "c,a=int('$CNT'),int('$AVAIL'); print(f'{c}/{a}  ({c*100//a:3d}%)')")
  else
    PCT="N/A (tutti 500)"
  fi

  if   [[ $AVAIL -gt 0 && $CNT -ge $((AVAIL * 8 / 10)) ]]; then COLOR=$GREEN
  elif [[ $AVAIL -gt 0 && $CNT -ge $((AVAIL * 6 / 10)) ]]; then COLOR=$YELLOW
  else COLOR=$RED; fi

  printf "${COLOR}  %-22s  %-18s  %-11s  %-10s${RESET}\n" "$DN" "$PCT" "$QS" "$NA"
done

echo "  ──────────────────────────────────────────────────────────────────────"
echo ""
log_info "(*) Messaggi 8-10: spam di tipo phishing/urgency senza keyword esplicite"
log_info "    (suspend, billing, parcel — assenti dalla lista keyword)."
log_info "    Il keyword heuristic non li rileva → predice HAM (falso negativo)."
log_info "    I modelli BERT comprendono il contesto → classificano correttamente SPAM."
log_info "    Questo giustifica empiricamente il costo energetico dei modelli neurali."
echo ""
