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
# TextSummarization — 4 variants on the Pareto front
#
#  Variant      Model                      Params  ROUGE-L  Energy/inv (estimate)
#  bart-large   facebook/bart-large-cnn    400M    ~40.9    ~1.200 J
#  distilbart   sshleifer/distilbart-12-6  306M    ~38.6    ~0.450 J
#  t5-small     google/t5-small             60M    ~30.1    ~0.080 J
#  extractive   TF-IDF sentence scoring      0M    ~23.0    ~0.003 J
#
# NOTE: invocation_joule values are initial estimates.
#       Run profile_variants.py --batch-json to replace with measured values.
# -------------------------------------------------------

log_info "Creating TS-bart-large (BART-large-cnn, quality_score=0.960)"
$SERVERLEDGE create \
  --function TS-bart-large \
  --logical-name TextSummarization \
  --runtime python-ml \
  --src ../../examples/Tesi/TextSummarization/TSSumBartLarge.py \
  --handler TSSumBartLarge.handler \
  --memory 4096 \
  --variant-id bart-large
log_ok "TS-bart-large created"

log_info "Creating TS-distilbart (DistilBART-cnn-12-6, quality_score=0.910)"
$SERVERLEDGE create \
  --function TS-distilbart \
  --logical-name TextSummarization \
  --runtime python-ml \
  --src ../../examples/Tesi/TextSummarization/TSSumDistilBart.py \
  --handler TSSumDistilBart.handler \
  --memory 3072 \
  --variant-id distilbart
log_ok "TS-distilbart created"

log_info "Creating TS-t5-small (T5-small, quality_score=0.760)"
$SERVERLEDGE create \
  --function TS-t5-small \
  --logical-name TextSummarization \
  --runtime python-ml \
  --src ../../examples/Tesi/TextSummarization/TSSumT5Small.py \
  --handler TSSumT5Small.handler \
  --memory 1024 \
  --variant-id t5-small
log_ok "TS-t5-small created"

log_info "Creating TS-extractive (TF-IDF extractive, quality_score=0.620, python310)"
$SERVERLEDGE create \
  --function TS-extractive \
  --logical-name TextSummarization \
  --runtime python310 \
  --src ../../examples/Tesi/TextSummarization/TSSumExtractive.py \
  --handler TSSumExtractive.handler \
  --memory 256 \
  --variant-id extractive
log_ok "TS-extractive created"

log_ok "All TextSummarization variants created"
log_ok "Workflow completed successfully"
