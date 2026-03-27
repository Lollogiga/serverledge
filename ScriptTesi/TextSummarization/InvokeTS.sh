#!/bin/bash
set -euo pipefail

BLUE="\e[34m"; GREEN="\e[32m"; YELLOW="\e[33m"; RED="\e[31m"; RESET="\e[0m"
log_info()  { echo -e "${BLUE}[INFO]${RESET}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET}  $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

SERVERLEDGE="../../bin/serverledge-cli"
[[ -x "$SERVERLEDGE" ]] || { log_error "serverledge-cli not found"; exit 1; }

# Short but non-trivial CNN/DailyMail-style paragraph used for all variants.
TEXT="The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris, France. \
It is named after the engineer Gustave Eiffel, whose company designed and built the tower from \
1887 to 1889. The tower is 330 metres tall and is the most visited paid monument in the world, \
attracting millions of visitors every year. It was originally intended as a temporary exhibit \
for the 1889 World Fair but became an enduring symbol of Paris and is now considered one of the \
greatest achievements in structural engineering."

log_info "Invoking TS-bart-large (BART-large-cnn)"
$SERVERLEDGE invoke --function TS-bart-large --param "text:$TEXT"
log_ok "TS-bart-large invocation completed"

log_info "Invoking TS-distilbart (DistilBART-cnn-12-6)"
$SERVERLEDGE invoke --function TS-distilbart --param "text:$TEXT"
log_ok "TS-distilbart invocation completed"

log_info "Invoking TS-t5-small (T5-small)"
$SERVERLEDGE invoke --function TS-t5-small --param "text:$TEXT"
log_ok "TS-t5-small invocation completed"

log_info "Invoking TS-extractive (TF-IDF extractive)"
$SERVERLEDGE invoke --function TS-extractive --param "text:$TEXT"
log_ok "TS-extractive invocation completed"

log_ok "All TextSummarization invocations completed"
