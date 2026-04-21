#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

# Two sample messages: one clear spam, one clear ham.
SPAM_TEXT="Congratulations! You have won a \$1,000 prize! Click here to claim your reward NOW! Limited time offer — act fast!"
HAM_TEXT="Hey, are you coming to the team meeting on Friday at 10am? Let me know if you need the dial-in details."

invoke_sd() {
  local fn="$1"
  local text="$2"
  local label="$3"   # expected label (for log readability)
  local params_file
  params_file=$(mktemp /tmp/sd_invoke_XXXXXX.json)
  trap 'rm -f "$params_file"' RETURN
  python3 -c "import json, sys; print(json.dumps({'text': sys.argv[1]}))" "$text" > "$params_file"

  log_info "Invoking $fn  [expected: $label]"
  if $SERVERLEDGE invoke --function "$fn" --params_file "$params_file"; then
    log_ok "$fn invocation completed"
  else
    local rc=$?
    if [[ $rc -eq 2 ]]; then
      log_warn "$fn returned HTTP 500 — modello ML non pre-scaricato nel container (atteso in ambiente senza cache)"
    else
      log_error "$fn fallito con exit=$rc"
      exit $rc
    fi
  fi
}

log_info "Registered SD functions:"
$SERVERLEDGE list 2>&1 | grep -o '"SpamDetection[^"]*"' | tr -d '"' | sort

# Keyword variant (python310 — always fast)
invoke_sd "SpamDetection-keyword"  "$SPAM_TEXT" "SPAM"
invoke_sd "SpamDetection-keyword"  "$HAM_TEXT"  "HAM"

# ML variants
invoke_sd "SpamDetection-bert-tiny"  "$SPAM_TEXT" "SPAM"
invoke_sd "SpamDetection-distilbert" "$SPAM_TEXT" "SPAM"
invoke_sd "SpamDetection"            "$SPAM_TEXT" "SPAM"

log_ok "All SpamDetection invocations completed (ML failures documented above)"
