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
# ImageClassification — 4 variants on the Pareto front
#
#  Variant            Model                        Params  Acc(ImageNet)  Energy/inv
#  vit-base           google/vit-base-patch16-224   86M    81.8%          ~1.455 J
#  efficientnet-b0    google/efficientnet_b0         5M    77.1%          ~0.450 J
#  mobilenet-v2       google/mobilenet_v2_1.0_224    3M    71.8%          ~0.229 J
#  mobilenet-v2-tiny  google/mobilenet_v2_0.35_224   2M    60.3%          ~0.080 J
# -------------------------------------------------------

# Creates ImageClassification (vit-base) + IC-efficientnet-b0 + IC-mobilenet-v2 + IC-mobilenet-v2-tiny automatically
log_info "Creating ImageClassification with all variants (--approximate)"
$SERVERLEDGE create \
  --function ImageClassification \
  --runtime python-ml \
  --src ../../variants/ImageClassification/ImageClassification.py \
  --handler ImageClassification.handler \
  --memory 2048 \
  --approximate

log_ok "All ImageClassification variants created (ImageClassification / ImageClassification-efficientnet-b0 / ImageClassification-mobilenet-v2 / ImageClassification-mobilenet-v2-tiny)"
