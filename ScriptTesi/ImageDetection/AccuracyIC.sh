#!/bin/bash
# =============================================================================
# AccuracyIC.sh — Binary accuracy test for ImageClassification variants
# =============================================================================
# Tests each IC variant on 3 labeled images from ./TestImage/.
# Checks if the returned label contains the expected keyword (substring match,
# case-insensitive). The result for each prediction is ✓ or ✗.
#
# Expected labels (ImageNet vocabulary, substring match):
#   GoldenRetriever.jpg  →  label contains "golden"
#   Husky.jpeg           →  label contains "husky" | "malamute" | "siberian" | "alaskan"
#   Airplane.jpg         →  label contains "airliner" | "airplane" | "plane" | "jet"
#
# Published ImageNet Top-1 accuracy for each model is shown alongside the
# measured per-sample correctness. This grounds the abstract quality_score
# values in concrete benchmark numbers.
#
# Usage:  ./AccuracyIC.sh
# =============================================================================

set -euo pipefail
cd "$(dirname "$0")"

SERVERLEDGE="../../bin/serverledge-cli"
IMG_DIR="./TestImage"
TMPFILE=$(mktemp /tmp/ic_acc_XXXXXX.json)
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
# TEST SET — 3 images with ground-truth keyword patterns
# =============================================================================
declare -a IMAGES=(
  "$IMG_DIR/GoldenRetriever.jpg"
  "$IMG_DIR/Husky.jpeg"
  "$IMG_DIR/Airplane.jpg"
)
declare -a LABELS=("Golden Retriever" "Husky" "Airplane")

# Regex patterns checked against the returned label (case-insensitive)
declare -a PATTERNS=(
  "golden"
  "husky|malamute|siberian|alaskan"
  "airliner|airplane|plane|jet|aircraft"
)

N_IMAGES=${#IMAGES[@]}

# =============================================================================
# VARIANTS — ordered lightest → heaviest
# =============================================================================
declare -a VARIANTS=(
  "ImageClassification-mobilenet-v2-tiny"
  "ImageClassification-mobilenet-v2"
  "ImageClassification-efficientnet-b0"
  "ImageClassification"
)
declare -a DISPLAY_NAMES=(
  "MobileNet-V2-tiny  (60.3% ImageNet)"
  "MobileNet-V2       (71.8% ImageNet)"
  "EfficientNet-B0    (77.1% ImageNet)"
  "ViT-base           (81.8% ImageNet)"
)
declare -a QUALITY_SCORES=("0.50" "0.65" "0.80" "0.95")

N_VARIANTS=${#VARIANTS[@]}

declare -a CORRECT=()
declare -a NA_COUNT=()

# =============================================================================
# MAIN LOOP
# =============================================================================
echo ""
log_info "Avvio accuracy test — ${N_IMAGES} immagini etichettate (GoldenRetriever, Husky, Airplane)"
log_info "Correttezza: match substring case-insensitive tra il label predetto e il pattern atteso."
echo ""

for vi in "${!VARIANTS[@]}"; do
  FN="${VARIANTS[$vi]}"
  DN="${DISPLAY_NAMES[$vi]}"
  correct=0
  na=0

  log_info "────────────────────────────────────────────────────────────"
  log_info "Variante: $FN  [$DN]"

  for ii in "${!IMAGES[@]}"; do
    IMG="${IMAGES[$ii]}"
    LBL="${LABELS[$ii]}"
    PAT="${PATTERNS[$ii]}"

    if [[ ! -f "$IMG" ]]; then
      log_warn "  $LBL: file non trovato ($IMG) → skip"
      na=$((na+1))
      continue
    fi

    # Encode image as base64 and build params JSON
    B64=$(base64 -w 0 "$IMG")
    python3 -c "import json, sys; print(json.dumps({'image_base64': sys.argv[1]}))" "$B64" > "$TMPFILE"

    # Invoke and capture output
    OUTPUT=$("$SERVERLEDGE" invoke \
      --function "$FN" \
      --params_file "$TMPFILE" \
      --ret_output 2>/dev/null) || {
      rc=$?
      if [[ $rc -eq 2 ]]; then
        log_warn "  $LBL: HTTP 500 — modello non pre-scaricato nel container → N/A"
        na=$((na+1))
        continue
      fi
      echo -e "${RED}[ERROR]${RESET} $FN su $LBL fallito con exit=$rc"; exit $rc
    }

    # Extract "label" field from the JSON response
    PRED=$(echo "$OUTPUT" | python3 -c "
import json, sys, re
text = sys.stdin.read()
m = re.search(r'\{[^{}]*\"label\"[^{}]*\}', text)
if m:
    try:
        d = json.loads(m.group())
        print(d.get('label', 'UNKNOWN'))
        sys.exit(0)
    except Exception:
        pass
print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

    # Binary check: does the predicted label contain the expected keyword?
    MATCH=$(python3 -c "
import re, sys
pred = sys.argv[1].lower()
pattern = sys.argv[2]
print('YES' if re.search(pattern, pred) else 'NO')
" "$PRED" "$PAT")

    if [[ "$MATCH" == "YES" ]]; then
      log_ok  "  $LBL  [atteso: $PAT]  →  \"$PRED\"  ✓"
      correct=$((correct+1))
    else
      log_warn "  $LBL  [atteso: $PAT]  →  \"$PRED\"  ✗"
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
echo -e "${BOLD}  RISULTATI CLASSIFICAZIONE — ImageClassification  (${N_IMAGES} immagini test)${RESET}"
echo -e "${BOLD}══════════════════════════════════════════════════════════════════════${RESET}"
printf "${BOLD}  %-36s  %-18s  %-11s  %-10s${RESET}\n" \
  "Modello (ImageNet Top-1)" "Corretti / Acc.%" "Q-score" "N/A (500)"
echo "  ──────────────────────────────────────────────────────────────────────"

for vi in "${!VARIANTS[@]}"; do
  DN="${DISPLAY_NAMES[$vi]}"
  CNT="${CORRECT[$vi]}"
  NA="${NA_COUNT[$vi]}"
  QS="${QUALITY_SCORES[$vi]}"
  AVAIL=$((N_IMAGES - NA))

  if [[ $AVAIL -gt 0 ]]; then
    PCT=$(python3 -c "c,a=int('$CNT'),int('$AVAIL'); print(f'{c}/{a}  ({c*100//a:3d}%)')")
  else
    PCT="N/A (tutti 500)"
  fi

  if   [[ $AVAIL -eq $N_IMAGES && $CNT -eq $AVAIL ]]; then COLOR=$GREEN
  elif [[ $AVAIL -gt 0 && $CNT -ge $((AVAIL * 2 / 3)) ]]; then COLOR=$YELLOW
  else COLOR=$RED; fi

  printf "${COLOR}  %-36s  %-18s  %-11s  %-10s${RESET}\n" "$DN" "$PCT" "$QS" "$NA"
done

echo "  ──────────────────────────────────────────────────────────────────────"
echo ""
log_info "Correttezza definita come substring match (case-insensitive):"
log_info "  GoldenRetriever →  contiene 'golden'"
log_info "  Husky           →  contiene 'husky', 'malamute', 'siberian' o 'alaskan'"
log_info "  Airplane        →  contiene 'airliner', 'airplane', 'plane' o 'jet'"
log_info ""
log_info "I modelli più pesanti (ViT-base, EfficientNet-B0) sono attesi più accurati"
log_info "su immagini ambigue. Questa misura diretta giustifica il quality_score nei"
log_info "JSON delle varianti e il tradeoff energia/qualità del selettore Pareto."
echo ""
