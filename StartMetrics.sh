#!/bin/bash
set -euo pipefail

# =========================
# Colori
# =========================
BLUE="\e[34m"
GREEN="\e[32m"
YELLOW="\e[33m"
RED="\e[31m"
RESET="\e[0m"

log_info() {
  echo -e "${BLUE}[INFO]${RESET} $1"
}

log_ok() {
  echo -e "${GREEN}[OK]${RESET} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${RESET} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${RESET} $1" >&2
}

# =========================
# Wrapper docker (sudo-safe)
# =========================
d() {
  if docker info >/dev/null 2>&1; then
    docker "$@"
  else
    sudo docker "$@"
  fi
}

# =========================
# Stop & remove containers
# =========================
log_info "Stopping existing containers (if any)"
containers="$(d ps -aq || true)"

if [[ -n "$containers" ]]; then
  d stop $containers || true
  d rm $containers || true
  log_ok "Containers stopped and removed"
else
  log_warn "No containers to stop/remove"
fi

# =========================
# Start Kepler
# =========================
log_info "Starting Kepler"
d run -d \
  --name kepler \
  --network host \
  --privileged \
  --pid=host \
  -v /sys:/sys \
  -v /proc:/proc \
  -v /var/run/docker.sock:/var/run/docker.sock \
  quay.io/sustainable_computing_io/kepler:latest

log_ok "Kepler started"

# =========================
# Start Prometheus
# =========================
if [[ ! -f "$(pwd)/prometheus.yml" ]]; then
  log_error "prometheus.yml not found in current directory"
  exit 1
fi

log_info "Starting Prometheus"
d run -d \
  --name prometheus \
  --network host \
  -v "$(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml" \
  prom/prometheus

log_ok "Prometheus started"

# =========================
# Start Grafana
# =========================
log_info "Starting Grafana"
d run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana

log_ok "Grafana started (http://localhost:3000)"

# =========================
# Start etcd
# =========================
log_info "Starting etcd"
d run -d \
  --name etcd \
  -p 2379:2379 \
  -p 2380:2380 \
  quay.io/coreos/etcd:v3.5.15 \
  /usr/local/bin/etcd \
  --name my-etcd-1 \
  --data-dir /etcd-data \
  --advertise-client-urls http://0.0.0.0:2379 \
  --listen-client-urls http://0.0.0.0:2379

log_ok "etcd started"

# =========================
# Done
# =========================
