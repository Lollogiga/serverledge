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

log_info()  { echo -e "${BLUE}[INFO]${RESET} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${RESET} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET} $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1" >&2; }

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
# Stop container if exists
# =========================
stop_if_running() {
  local name="$1"
  if d ps -a --format '{{.Names}}' | grep -q "^${name}$"; then
    log_info "Stopping existing container: $name"
    d stop "$name" || true
    d rm "$name" || true
  fi
}

# =========================
# Check prometheus.yml
# =========================
if [[ ! -f "$(pwd)/prometheus.yml" ]]; then
  log_error "prometheus.yml not found in current directory"
  exit 1
fi

# =========================
# Start Kepler
# =========================
stop_if_running kepler

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
stop_if_running prometheus

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
stop_if_running grafana

log_info "Starting Grafana"
d run -d \
  --name grafana \
  --network host \
  grafana/grafana

log_ok "Grafana started (http://localhost:3000)"

# =========================
# Start etcd (PERSISTENT)
# =========================
stop_if_running etcd

log_info "Starting etcd"
d run -d \
  --name etcd \
  -p 2379:2379 \
  -p 2380:2380 \
  -v etcd-data:/etcd-data \
  quay.io/coreos/etcd:v3.5.15 \
  /usr/local/bin/etcd \
  --name my-etcd-1 \
  --data-dir /etcd-data \
  --advertise-client-urls http://0.0.0.0:2379 \
  --listen-client-urls http://0.0.0.0:2379

log_ok "etcd started"

# =========================
# Start InfluxDB (PERSISTENT)
# =========================
stop_if_running influxdb

log_info "Starting InfluxDB"
d run -d \
  --name influxdb \
  -p 8086:8086 \
  -v influxdb-data:/var/lib/influxdb2 \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=admin123 \
  -e DOCKER_INFLUXDB_INIT_ORG=serverledge \
  -e DOCKER_INFLUXDB_INIT_BUCKET=serverledge-energy \
  -e DOCKER_INFLUXDB_INIT_RETENTION=0 \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=serverledge-token \
  influxdb:2.7

log_ok "InfluxDB started (http://localhost:8086)"

# =========================
# Done
# =========================
log_ok "Infrastructure started successfully"