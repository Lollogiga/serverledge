#!/bin/bash
# Batch-profile all TextSummarization variants and auto-update the JSON.
# Usage: ./ProfileTS.sh

set -euo pipefail
cd "$(dirname "$0")"

# Credenziali InfluxDB (da StartMetrics.sh)
INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

TEXT="The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris, France. \
It is named after the engineer Gustave Eiffel, whose company designed and built the tower from \
1887 to 1889. The tower is 330 metres tall and is the most visited paid monument in the world, \
attracting millions of visitors every year. It was originally intended as a temporary exhibit \
for the 1889 World Fair but became an enduring symbol of Paris and is now considered one of the \
greatest achievements in structural engineering."

/usr/bin/python3 ../profile_variants.py \
  --batch-json ../../variants/TextSummarization/TextSummarization.json \
  --function-prefix TS \
  --n-warmup 2 \
  --n-samples 20 \
  --param "text:$TEXT" \
  --influx-url "$INFLUX_URL" \
  --influx-token "$INFLUX_TOKEN" \
  --influx-org "$INFLUX_ORG" \
  --influx-bucket "$INFLUX_BUCKET" \
  --serverledge-cli ../../bin/serverledge-cli
