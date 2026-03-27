#!/bin/bash
# Batch-profile tutte le varianti ImageClassification e aggiorna automaticamente il JSON.
# Uso: ./ProfileIC.sh
# Nota: le varianti ImageClassification richiedono un parametro "image_url" o "image_b64".
#       Passa un'immagine di test appropriata.

set -euo pipefail
cd "$(dirname "$0")"

# Credenziali InfluxDB (da StartMetrics.sh)
INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

/usr/bin/python3 ../profile_variants.py \
  --batch-json ../../variants/ImageClassification/ImageClassification.json \
  --function-prefix IC \
  --n-warmup 2 \
  --n-samples 20 \
  --influx-url "$INFLUX_URL" \
  --influx-token "$INFLUX_TOKEN" \
  --influx-org "$INFLUX_ORG" \
  --influx-bucket "$INFLUX_BUCKET" \
  --serverledge-cli ../../bin/serverledge-cli
