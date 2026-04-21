#!/bin/bash
set -euo pipefail

SERVERLEDGE="../../bin/serverledge-cli"

YELLOW="\e[33m"; RESET="\e[0m"
sep() { echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"; }

# ─── InfluxDB ─────────────────────────────────────────────────
INFLUX_URL="http://localhost:8086"
INFLUX_TOKEN="serverledge-token"
INFLUX_ORG="serverledge"
INFLUX_BUCKET="serverledge-energy"

influx_write() {
    local zone="$1" json="$2"
    local fn selected ci qw energy err pareto ts line
    fn=$(echo "$json"       | jq -r '.variant_scheduling.logical_name           // "unknown"')
    selected=$(echo "$json" | jq -r '.variant_scheduling.selected_function      // "none"')
    ci=$(echo "$json"       | jq -r '.variant_scheduling.carbon_intensity_gco2  // 0')
    qw=$(echo "$json"       | jq -r '.variant_scheduling.quality_weight         // 0')
    energy=$(echo "$json"   | jq -r '.variant_scheduling.estimated_energy_joule // 0')
    err=$(echo "$json"      | jq -r '.variant_scheduling.error_estimate         // 0')
    pareto=$(echo "$json"   | jq -c '.variant_scheduling.pareto_front // []' \
                              | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
    ts=$(date +%s%N)

    line="pareto_scheduling,function=${fn},zone=${zone} \
selected_variant=\"${selected}\",\
carbon_intensity=${ci},\
quality_weight=${qw},\
estimated_energy_joule=${energy},\
error_estimate=${err},\
pareto_front=\"${pareto}\" ${ts}"

    curl -s -o /dev/null \
        -XPOST "${INFLUX_URL}/api/v2/write?org=${INFLUX_ORG}&bucket=${INFLUX_BUCKET}&precision=ns" \
        -H "Authorization: Token ${INFLUX_TOKEN}" \
        -H "Content-Type: text/plain; charset=utf-8" \
        --data-raw "$line" || true
}

invoke_zone() {
    local zone="$1"; shift
    local resp
    sep; echo "Zona: ${zone:-config default}"
    resp=$("$SERVERLEDGE" invoke --function LnHarmonic "$@" 2>/dev/null)
    echo "$resp" | jq 'del(.variant_scheduling.pareto_front)'
    influx_write "${zone:-default}" "$resp"
    sleep 1
}

# ─────────────────────────────────────────────────────────────
# CLEANUP
# ─────────────────────────────────────────────────────────────
echo "Cleaning existing functions..."
$SERVERLEDGE list 2>/dev/null \
  | tr -d '[]",' \
  | sed 's/^[[:space:]]*//' \
  | sed '/^$/d' \
  | while read -r fn; do
      $SERVERLEDGE delete --function "$fn" || true
    done

# ─────────────────────────────────────────────────────────────
# CREATE
# ─────────────────────────────────────────────────────────────
$SERVERLEDGE create \
  --function LnHarmonic \
  --runtime python310 \
  --src ../../variants/LnHarmonic/lnHarmonic.py \
  --handler lnHarmonic.handler \
  --memory 128 \
  --output "y:Float" \
  --approximate \
  --shareContainer

sleep 2

# ─────────────────────────────────────────────────────────────
# PREWARM
# ─────────────────────────────────────────────────────────────
$SERVERLEDGE prewarm --function LnHarmonic-n1000 --count 1
$SERVERLEDGE prewarm --function LnHarmonic       --count 1

sleep 3

# ─────────────────────────────────────────────────────────────
# INVOCAZIONI per CI-zone
# Attese (convergenza O(1/N), speculare a PiLeibniz):
#   NO-NO5  ~20  gCO2/kWh  → λ alto   → base (N=1M, err≈0)
#   FR      ~60  gCO2/kWh  → λ alto   → base / n200000
#   IT-NO   ~200 gCO2/kWh  → λ~0.63  → n50000 / n100000
#   DE      ~350 gCO2/kWh  → λ~0.55  → n10000 / n50000
#   PL      ~750 gCO2/kWh  → λ basso → n1000
# ─────────────────────────────────────────────────────────────
invoke_zone ""
invoke_zone "NO-NO5" --ci-zone NO-NO5
invoke_zone "FR"     --ci-zone FR
invoke_zone "IT-NO"  --ci-zone IT-NO
invoke_zone "DE"     --ci-zone DE
invoke_zone "PL"     --ci-zone PL
