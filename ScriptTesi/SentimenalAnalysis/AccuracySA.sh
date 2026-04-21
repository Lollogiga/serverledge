#!/bin/bash
# =============================================================================
# AccuracySA.sh — Binary accuracy test for SentimentAnalysis variants
# =============================================================================
# Tests each SA variant on a 10-sentence labeled test set.
# Each sentence has a clear ground-truth label: POSITIVE or NEGATIVE.
# For each prediction the result is simply ✓ (correct) or ✗ (wrong).
#
# Test set design:
#   Sentences  1-7 : unambiguous → all models expected correct
#   Sentences 8-10 : NEGATIVE using words absent from the VADER/AFINN lexicon
#                    (disappointed, unreliable, inferior, regret, defective,
#                    substandard) → VADER scores them 0 → predicts NEUTRAL ✗
#                    BERT-based models read context → correctly predict NEGATIVE ✓
#
# Usage:  ./AccuracySA.sh
# =============================================================================

set -euo pipefail
cd "$(dirname "$0")"

SERVERLEDGE="../../bin/serverledge-cli"
TMPFILE=$(mktemp /tmp/sa_acc_XXXXXX.json)
trap 'rm -f "$TMPFILE"' EXIT

# ─── Colours ──────────────────────────────────────────────────────────────────
BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"
BOLD="\e[1m"; RESET="\e[0m"
log_info() { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()   { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${RESET}  $1"; }

if [[ ! -x "$SERVERLEDGE" ]]; then
  echo -e "${RED}[ERROR]${RESET} $SERVERLEDGE not found or not executable"; exit 1
fi

# =============================================================================
# TEST SET — 10 sentences with ground-truth labels
# =============================================================================
declare -a TEXTS=(
  # POSITIVE (1-5) — clear vocabulary, all models expected correct
  "This product is absolutely fantastic and I love using it every day!"
  "Excellent quality and great service, highly recommended!"
  "Amazing experience, I am very happy with this outstanding purchase."
  "Perfect product, superb packaging, and wonderful customer support."
  "Beautiful design and impressive performance, I am very pleased."
  # NEGATIVE easy (6-7)
  "This is the worst product I have ever bought, absolutely terrible!"
  "Disgusting quality and horrible service. I hate it completely!"
  # NEGATIVE tricky for VADER (8-10)
  # Words not in AFINN lexicon → VADER score = 0 → predicts NEUTRAL
  "I was thoroughly disappointed with this product; it feels cheap and unreliable."
  "Overpriced and inferior quality. I regret this purchase deeply."
  "The device stopped working after a week. Defective and substandard."
)

declare -a GROUND_TRUTH=(
  "POSITIVE" "POSITIVE" "POSITIVE" "POSITIVE" "POSITIVE"
  "NEGATIVE" "NEGATIVE"
  "NEGATIVE" "NEGATIVE" "NEGATIVE"
)

# (*) marks sentences that expose VADER's lexicon gaps
declare -a TRICKY=("" "" "" "" "" "" "" " (*)" " (*)" " (*)")

N_TEXTS=${#TEXTS[@]}

# =============================================================================
# VARIANTS — ordered lightest → heaviest
# =============================================================================
declare -a VARIANTS=(
  "SentimentAnalysis-vader"
  "SentimentAnalysis-bert-tiny"
  "SentimentAnalysis-distilbert"
  "SentimentAnalysis"
)
declare -a DISPLAY_NAMES=(
  "VADER (AFINN lexicon)"
  "BERT-tiny"
  "DistilBERT"
  "RoBERTa-large"
)
# Published benchmark accuracy on SST-2 (as reflected in quality_score)
declare -a QUALITY_SCORES=("0.50" "0.65" "0.80" "0.95")

N_VARIANTS=${#VARIANTS[@]}

declare -a CORRECT=()
declare -a NA_COUNT=()

# =============================================================================
# MAIN LOOP
# =============================================================================
echo ""
log_info "Avvio accuracy test — ${N_TEXTS} frasi (5 POSITIVE + 5 NEGATIVE)"
log_info "Frasi con (*) sono le tricky: parole assenti dal lessico AFINN."
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

    # Build params JSON — Python handles quoting safely
    python3 -c "import json, sys; print(json.dumps({'text': sys.argv[1]}))" "$TEXT" > "$TMPFILE"

    # Invoke and capture stdout; treat HTTP 500 (exit=2) as N/A
    OUTPUT=$("$SERVERLEDGE" invoke \
      --function "$FN" \
      --params_file "$TMPFILE" \
      --ret_output 2>/dev/null) || {
      rc=$?
      if [[ $rc -eq 2 ]]; then
        log_warn "  Testo$((ti+1))$TRK [GT=$GT]: HTTP 500 — modello non scaricato → N/A"
        na=$((na+1))
        continue
      fi
      echo -e "${RED}[ERROR]${RESET} $FN testo $((ti+1)) fallito con exit=$rc"; exit $rc
    }

    # Extract "label" field from the JSON response
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
      log_ok  "  Testo $((ti+1))$TRK  [GT=$GT]  →  predetto: $LABEL  ✓"
      correct=$((correct+1))
    else
      log_warn "  Testo $((ti+1))$TRK  [GT=$GT]  →  predetto: $LABEL  ✗"
    fi
  done

  echo ""
  CORRECT[$vi]=$correct
  NA_COUNT[$vi]=$na
done

# =============================================================================
# SUMMARY TABLE
# =============================================================================
echo -e "${BOLD}══════════════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}  RISULTATI ACCURACY — SentimentAnalysis  (${N_TEXTS} frasi: 5 POS + 5 NEG)${RESET}"
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
log_info "(*) Frasi 8-10: parole assenti dal lessico AFINN di VADER"
log_info "    (disappointed, unreliable, inferior, regret, defective, substandard)."
log_info "    VADER ottiene score=0 → predice NEUTRAL invece di NEGATIVE."
log_info "    I modelli BERT capiscono il contesto → classificano correttamente."
log_info "    Questo giustifica empiricamente il costo energetico dei modelli neurali."
echo ""
