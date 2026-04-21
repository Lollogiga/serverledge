#!/bin/bash
# Batch-profile all SpamDetection variants and auto-update the JSON.
# Usage: ./ProfileSD.sh

set -euo pipefail
cd "$(dirname "$0")"

# Credenziali InfluxDB (da StartMetrics.sh)
INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

SPAM_TEXT="Congratulations! You have won a \$1,000 prize! Click here to claim your reward NOW! Limited time offer — act fast!"

/usr/bin/python3 ../profile_variants.py \
  --batch-json ../../variants/SpamDetection/SpamDetection.json \
  --function-prefix SD \
  --n-warmup 2 \
  --n-samples 20 \
  --param "text:$SPAM_TEXT" \
  --influx-url "$INFLUX_URL" \
  --influx-token "$INFLUX_TOKEN" \
  --influx-org "$INFLUX_ORG" \
  --influx-bucket "$INFLUX_BUCKET" \
  --serverledge-cli ../../bin/serverledge-cli
