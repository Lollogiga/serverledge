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
# SpamDetection — 4 variants on the Pareto front
#
#  Variant      Model                                 Params  Acc     Energy/inv (estimate)
#  roberta-spam mshenoda/roberta-spam                 125M    99.5%   ~0.085 J
#  distilbert   Falconsai/spam_classification          67M    97.0%   ~0.035 J
#  bert-tiny    mrm8488/bert-tiny-finetuned-sms-spam   4.4M   95.0%   ~0.010 J
#  keyword      keyword-heuristic (pure Python)          0M   72.0%   ~0.002 J
#
# NOTE: invocation_joule values are initial estimates.
#       Run ProfileSD.sh to replace with measured values.
# -------------------------------------------------------

log_info "Creating SpamDetection with all variants (--approximate)"
$SERVERLEDGE create \
  --function SpamDetection \
  --runtime python-ml \
  --src ../../variants/SpamDetection/SDHeavy.py \
  --handler SDHeavy.handler \
  --memory 2048 \
  --approximate

log_ok "All SpamDetection variants created (SpamDetection / SpamDetection-distilbert / SpamDetection-bert-tiny / SpamDetection-keyword)"
log_ok "Workflow completed successfully"
