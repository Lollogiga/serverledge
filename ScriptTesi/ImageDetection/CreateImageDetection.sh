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

log_info "Creating IC-vit-base (ViT base, quality_score=0.818)"
$SERVERLEDGE create \
  --function IC-vit-base \
  --logical-name ImageClassification \
  --runtime python-ml \
  --src ../../examples/Tesi/ImageClassification/ImageClassification.py \
  --handler ImageClassification.handler \
  --memory 2048 \
  --variant-id vit-base
log_ok "IC-vit-base created"

log_info "Creating IC-efficientnet-b0 (EfficientNet-B0, quality_score=0.771)"
$SERVERLEDGE create \
  --function IC-efficientnet-b0 \
  --logical-name ImageClassification \
  --runtime python-ml \
  --src ../../examples/Tesi/ImageClassification/ImageClassificationEfficientNetB0.py \
  --handler ImageClassificationEfficientNetB0.handler \
  --memory 2048 \
  --variant-id efficientnet-b0
log_ok "IC-efficientnet-b0 created"

log_info "Creating IC-mobilenet-v2 (MobileNetV2 1.0, quality_score=0.718)"
$SERVERLEDGE create \
  --function IC-mobilenet-v2 \
  --logical-name ImageClassification \
  --runtime python-ml \
  --src ../../examples/Tesi/ImageClassification/ImageClassificationLight.py \
  --handler ImageClassificationLight.handler \
  --memory 1024 \
  --variant-id mobilenet-v2
log_ok "IC-mobilenet-v2 created"

log_info "Creating IC-mobilenet-v2-tiny (MobileNetV2 0.35, quality_score=0.603)"
$SERVERLEDGE create \
  --function IC-mobilenet-v2-tiny \
  --logical-name ImageClassification \
  --runtime python-ml \
  --src ../../examples/Tesi/ImageClassification/ImageClassificationMobileNetTiny.py \
  --handler ImageClassificationMobileNetTiny.handler \
  --memory 1024 \
  --variant-id mobilenet-v2-tiny
log_ok "IC-mobilenet-v2-tiny created"

log_ok "All ImageClassification variants created"
