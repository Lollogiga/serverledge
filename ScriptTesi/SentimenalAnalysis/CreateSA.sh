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
# SentimentAnalysis — 4 variants on the Pareto front
#
#  Variant          Model                         Params  Acc(SST-2)  Energy/inv
#  roberta-large    siebert/...roberta-large       355M    96.4%       ~0.092 J
#  distilbert       distilbert-base-uncased-s2      67M    91.3%       ~0.040 J
#  bert-tiny        mrm8488/bert-tiny-sst2           4M    84.0%       ~0.012 J
#  vader            AFINN lexicon (pure Python)      0M    70.0%       ~0.003 J
# -------------------------------------------------------

log_info "Creating SA-roberta-large (RoBERTa-large, quality_score=0.964)"
$SERVERLEDGE create \
  --function SA-roberta-large \
  --logical-name SentimentAnalysis \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SAHeavy.py \
  --handler SAHeavy.handler \
  --memory 2048 \
  --variant-id roberta-large
log_ok "SA-roberta-large created"

log_info "Creating SA-distilbert (DistilBERT, quality_score=0.913)"
$SERVERLEDGE create \
  --function SA-distilbert \
  --logical-name SentimentAnalysis \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SADistilBERT.py \
  --handler SADistilBERT.handler \
  --memory 2048 \
  --variant-id distilbert
log_ok "SA-distilbert created"

log_info "Creating SA-bert-tiny (BERT-tiny, quality_score=0.840)"
$SERVERLEDGE create \
  --function SA-bert-tiny \
  --logical-name SentimentAnalysis \
  --runtime python-ml \
  --src ../../examples/Tesi/SentimentAnalysis/SABertTiny.py \
  --handler SABertTiny.handler \
  --memory 1024 \
  --variant-id bert-tiny
log_ok "SA-bert-tiny created"

log_info "Creating SA-vader (AFINN lexicon, quality_score=0.700, python310)"
$SERVERLEDGE create \
  --function SA-vader \
  --logical-name SentimentAnalysis \
  --runtime python310 \
  --src ../../examples/Tesi/SentimentAnalysis/SAvader.py \
  --handler SAvader.handler \
  --memory 256 \
  --variant-id vader
log_ok "SA-vader created"

log_ok "All SentimentAnalysis variants created"
# =====================================================
log_ok "Workflow completed successfully"
