#!/bin/bash
set -euo pipefail

# =====================================================
# COLORI PER I LOG
# =====================================================
BLUE="\e[34m"
GREEN="\e[32m"
YELLOW="\e[33m"
RED="\e[31m"
RESET="\e[0m"

log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

# =====================================================
# SERVERLEDGE CLI
# =====================================================
SERVERLEDGE="../../bin/serverledge-cli"

if [[ ! -x "$SERVERLEDGE" ]]; then
  log_error "serverledge-cli not found or not executable"
  exit 1
fi

# =====================================================
# CLEANUP FUNZIONI SERVERLEDGE
# =====================================================
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

# =====================================================
# CREATE – FUNZIONE BASE (VARIANTI AUTO)
# =====================================================
log_info "Creating ImageClassification base function (approximate enabled)"

$SERVERLEDGE create \
  --function ImageClassification \
  --runtime python-ml \
  --src ../../examples/Tesi/ImageClassification/ImageClassification.py \
  --handler ImageClassification.handler \
  --input "image_base64:Text" \
  --output "label:Text,confidence:Float" \
  --memory 1024 \
  --approximate

log_ok "ImageClassification base function created (variants loaded automatically)"

sleep 3

# =====================================================
# IMMAGINI DI TEST
# =====================================================
IMAGES=(
  "./TestImage/GoldenRetriever.jpg"
  "./TestImage/Husky.jpeg"
  "./TestImage/Airplane.jpg"
)

B64_FILE="./TestImage/tmp.b64"
PARAMS_FILE="./TestImage/params.json"

# =====================================================
# TEST INVOCATIONS
# =====================================================
for IMAGE_FILE in "${IMAGES[@]}"; do

  if [[ ! -f "$IMAGE_FILE" ]]; then
    log_warn "Image file '$IMAGE_FILE' not found, skipping"
    continue
  fi

  IMAGE_NAME=$(basename "$IMAGE_FILE")
  log_info "Processing image: $IMAGE_NAME"

  # ---------------------------------------------------
  # BASE64
  # ---------------------------------------------------
  base64 -w 0 "$IMAGE_FILE" > "$B64_FILE"
  log_ok "Base64 encoded ($(wc -c < "$B64_FILE") bytes)"

  cat > "$PARAMS_FILE" <<EOF
{
  "image_base64": "$(cat "$B64_FILE")"
}
EOF

  # =====================================================
  # INVOKE 1 – LEGACY (NO APPROX)
  # =====================================================
  log_info "Invoking ImageClassification (legacy, base only)"

  $SERVERLEDGE invoke \
    --function ImageClassification \
    --params_file "$PARAMS_FILE" \
    --ret_output

  log_ok "Legacy invocation completed"
  sleep 2

  # =====================================================
  # INVOKE 2 – ALLOW APPROX
  # =====================================================
  log_info "Invoking ImageClassification with allowApprox"

  $SERVERLEDGE invoke \
    --function ImageClassification \
    --allowApprox \
    --params_file "$PARAMS_FILE" \
    --ret_output

  log_ok "allowApprox invocation completed"
  sleep 2

  # =====================================================
  # INVOKE 3 – ALLOW APPROX + BUDGET
  # =====================================================
  log_info "Invoking ImageClassification with allowApprox + energy budget"

  $SERVERLEDGE invoke \
    --function ImageClassification \
    --allowApprox \
    --maxEnergyJoule 1.35 \
    --params_file "$PARAMS_FILE" \
    --ret_output

  log_ok "Energy-constrained invocation completed"
  echo ""

done

# =====================================================
# DONE
# =====================================================
log_ok "Scheduler ImageClassification workflow completed successfully"
