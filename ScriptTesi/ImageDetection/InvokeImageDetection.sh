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
# IMMAGINI DI TEST (3)
# =====================================================
IMAGES=(
  "./TestImage/GoldenRetriever.jpg"
  "./TestImage/Husky.jpeg"
  "./TestImage/Airplane.jpg"
)

# =====================================================
# FUNZIONI / MODELLI
# =====================================================
FUNCTIONS=(
  "ImageClassification"        # MobileNetV2 (HQ)
  "ImageClassificationLight"   # MobileNetV2 0.5 (Low-energy)
)

# =====================================================
# FILE TEMP
# =====================================================
B64_FILE="./TestImage/tmp.b64"
PARAMS_FILE="./TestImage/params.json"


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

  # ---------------------------------------------------
  # PARAMS JSON
  # ---------------------------------------------------
  cat > "$PARAMS_FILE" <<EOF
{
  "image_base64": "$(cat "$B64_FILE")"
}
EOF

  # ---------------------------------------------------
  # INVOKE PER OGNI MODELLO
  # ---------------------------------------------------
  for FN in "${FUNCTIONS[@]}"; do
    log_info "Invoking $FN on $IMAGE_NAME"

    $SERVERLEDGE invoke \
      --function "$FN" \
      --params_file "$PARAMS_FILE" \
      --ret_output

    echo ""
  done

done

log_ok "All invocations completed"
