#!/bin/bash
# Batch-profile tutte le varianti SentimentAnalysis e aggiorna automaticamente il JSON.
# Uso: ./ProfileSA.sh

set -euo pipefail
cd "$(dirname "$0")"

# Credenziali InfluxDB (da StartMetrics.sh)
INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

/usr/bin/python3 ../profile_variants.py \
  --batch-json ../../variants/SentimentAnalysis/SentimentAnalysis.json \
  --function-prefix SA \
  --n-warmup 2 \
  --n-samples 30 \
  --param "text:The movie was absolutely fantastic, I loved every minute of it." \
  --influx-url "$INFLUX_URL" \
  --influx-token "$INFLUX_TOKEN" \
  --influx-org "$INFLUX_ORG" \
  --influx-bucket "$INFLUX_BUCKET" \
  --serverledge-cli ../../bin/serverledge-cli
