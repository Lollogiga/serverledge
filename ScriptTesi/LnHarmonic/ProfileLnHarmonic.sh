#!/bin/bash
# Batch-profile tutte le varianti LnHarmonic e aggiorna automaticamente il JSON.
# Uso: ./ProfileLnHarmonic.sh

set -euo pipefail
cd "$(dirname "$0")"

INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

/usr/bin/python3 ../profile_variants.py \
  --batch-json ../../variants/LnHarmonic/LnHarmonic.json \
  --function-prefix lnHarmonic \
  --n-warmup 3 \
  --n-samples 50 \
  --wait-after 90 \
  --poll-interval 5 \
  --influx-url "$INFLUX_URL" \
  --influx-token "$INFLUX_TOKEN" \
  --influx-org "$INFLUX_ORG" \
  --influx-bucket "$INFLUX_BUCKET" \
  --serverledge-cli ../../bin/serverledge-cli
