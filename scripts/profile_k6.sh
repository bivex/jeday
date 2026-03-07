#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PGBOUNCER_INI="$ROOT_DIR/pgbouncer.ini"
PGBOUNCER_INI_BACKUP=""

COMMAND="${1:-full}"
VUS="${VUS:-500}"
DURATION="${DURATION:-20s}"
LOAD_SCRIPT="${LOAD_SCRIPT:-load-test-reg-only.js}"
OUTPUT_BASE="${OUTPUT_BASE:-artifacts/profile-runs}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
RUN_DIR="${OUTPUT_DIR:-$OUTPUT_BASE/$TIMESTAMP}"

compose() { docker compose "$@"; }
compose_prof() { docker compose -f docker-compose.yml -f docker-compose.prof.yml "$@"; }

usage() {
  cat <<EOF
Usage: $(basename "$0") [reset|profile|full]

Commands:
  reset    Stop/remove compose containers, remove orphan profiler containers, networks and volumes.
  profile  Build/start stack, enable profiler, run k6, save artifacts to $RUN_DIR.
  full     reset + profile.

Environment overrides:
  VUS=500
  DURATION=20s
  LOAD_SCRIPT=load-test-reg-only.js
  OUTPUT_BASE=artifacts/profile-runs
  OUTPUT_DIR=<custom-output-dir>
EOF
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}

prepare_pgbouncer_ini() {
  PGBOUNCER_INI_BACKUP="$(mktemp "${TMPDIR:-/tmp}/jeday-pgbouncer.ini.XXXXXX")"
  cp "$PGBOUNCER_INI" "$PGBOUNCER_INI_BACKUP"

  cat > "$PGBOUNCER_INI" <<EOF
[pgbouncer]
listen_port = ${PGBOUNCER_PORT:-6432}
listen_addr = 0.0.0.0
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = ${PGBOUNCER_POOL_MODE:-transaction}
max_client_conn = ${PGBOUNCER_MAX_CLIENT_CONN:-10000}
default_pool_size = ${PGBOUNCER_DEFAULT_POOL_SIZE:-100}
min_pool_size = ${PGBOUNCER_MIN_POOL_SIZE:-0}
reserve_pool_size = ${PGBOUNCER_RESERVE_POOL_SIZE:-0}
ignore_startup_parameters = ${PGBOUNCER_IGNORE_STARTUP_PARAMETERS:-extra_float_digits,prefer_simple_protocol}

[databases]
auth_db = host=db port=5432 dbname=auth_db
EOF
}

restore_pgbouncer_ini() {
  if [[ -n "$PGBOUNCER_INI_BACKUP" && -f "$PGBOUNCER_INI_BACKUP" ]]; then
    mv "$PGBOUNCER_INI_BACKUP" "$PGBOUNCER_INI"
    PGBOUNCER_INI_BACKUP=""
  fi
}

cleanup_stack() {
  echo "==> Removing containers, networks and volumes"
  compose_prof down -v --remove-orphans || true
  compose down -v --remove-orphans || true
}

wait_for_health() {
  echo "==> Waiting for app health"
  for _ in $(seq 1 90); do
    if curl -sf http://localhost:8080/health >/dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "error: app did not become healthy" >&2
  return 1
}

collect_pg_stats() {
  local phase="$1"
  docker exec -e PGPASSWORD=secret jeday-db-1 \
    psql -U app_user -d auth_db -c \
    "SELECT datname, xact_commit, tup_inserted, blks_read, blks_hit FROM pg_stat_database WHERE datname='auth_db'; \
     SELECT wal_records, wal_bytes FROM pg_stat_wal; \
     SELECT checkpoints_timed, checkpoints_req, checkpoint_write_time, checkpoint_sync_time, buffers_checkpoint, buffers_clean, maxwritten_clean FROM pg_stat_bgwriter; \
     SELECT relname, idx_scan, n_tup_ins, n_live_tup, n_dead_tup FROM pg_stat_user_tables WHERE relname IN ('users','password_upgrade_queue','user_passwords') ORDER BY relname;" \
    > "$RUN_DIR/${phase}-pg-stats.txt"
}

collect_otel_samples() {
  compose_prof logs otelcol > "$RUN_DIR/otelcol.log" || true
  {
    printf 'auth-api samples: '
    grep -c '/app/auth-api' "$RUN_DIR/otelcol.log" || true
    printf 'auth-worker samples: '
    grep -c '/app/auth-worker' "$RUN_DIR/otelcol.log" || true
    printf 'postgres samples: '
    grep -c '/usr/local/bin/postgres' "$RUN_DIR/otelcol.log" || true
    printf 'k6 samples: '
    grep -c '/usr/bin/k6' "$RUN_DIR/otelcol.log" || true
  } > "$RUN_DIR/otel-sample-counts.txt"
}

write_tuning_snapshot() {
  cat > "$RUN_DIR/tunables.env" <<EOF
SERVER_PREFORK=${SERVER_PREFORK:-true}
APP_DB_POOL_MAX_CONNS=${APP_DB_POOL_MAX_CONNS:-128}
REGISTRATION_BATCH_SIZE=${REGISTRATION_BATCH_SIZE:-100}
REGISTRATION_BATCH_WAIT=${REGISTRATION_BATCH_WAIT:-10ms}
WORKER_DB_POOL_MAX_CONNS=${WORKER_DB_POOL_MAX_CONNS:-4}
WORKER_GOMAXPROCS=${WORKER_GOMAXPROCS:-1}
WORKER_GOMEMLIMIT=${WORKER_GOMEMLIMIT:-256MiB}
WORKER_INTERVAL=${WORKER_INTERVAL:-5s}
WORKER_UPGRADE_LIMIT=${WORKER_UPGRADE_LIMIT:-4}
PGBOUNCER_PORT=${PGBOUNCER_PORT:-6432}
PGBOUNCER_POOL_MODE=${PGBOUNCER_POOL_MODE:-transaction}
PGBOUNCER_MAX_CLIENT_CONN=${PGBOUNCER_MAX_CLIENT_CONN:-10000}
PGBOUNCER_DEFAULT_POOL_SIZE=${PGBOUNCER_DEFAULT_POOL_SIZE:-100}
PGBOUNCER_MIN_POOL_SIZE=${PGBOUNCER_MIN_POOL_SIZE:-0}
PGBOUNCER_RESERVE_POOL_SIZE=${PGBOUNCER_RESERVE_POOL_SIZE:-0}
PGBOUNCER_IGNORE_STARTUP_PARAMETERS=${PGBOUNCER_IGNORE_STARTUP_PARAMETERS:-extra_float_digits,prefer_simple_protocol}
POSTGRES_MAX_CONNECTIONS=${POSTGRES_MAX_CONNECTIONS:-256}
POSTGRES_SHARED_BUFFERS=${POSTGRES_SHARED_BUFFERS:-512MB}
POSTGRES_SYNCHRONOUS_COMMIT=${POSTGRES_SYNCHRONOUS_COMMIT:-off}
POSTGRES_FULL_PAGE_WRITES=${POSTGRES_FULL_PAGE_WRITES:-off}
POSTGRES_CHECKPOINT_TIMEOUT=${POSTGRES_CHECKPOINT_TIMEOUT:-15min}
POSTGRES_CHECKPOINT_COMPLETION_TARGET=${POSTGRES_CHECKPOINT_COMPLETION_TARGET:-0.95}
POSTGRES_WAL_BUFFERS=${POSTGRES_WAL_BUFFERS:-64MB}
POSTGRES_EFFECTIVE_IO_CONCURRENCY=${POSTGRES_EFFECTIVE_IO_CONCURRENCY:-32}
VUS=$VUS
DURATION=$DURATION
LOAD_SCRIPT=$LOAD_SCRIPT
EOF
}

run_profile() {
  mkdir -p "$RUN_DIR"

  echo "==> Starting app stack"
  prepare_pgbouncer_ini
  write_tuning_snapshot
  compose up --build -d
  wait_for_health

  echo "==> Starting otelcol + profiler"
  compose_prof up -d otelcol profiler

  echo "==> Capturing baseline stats"
  compose ps > "$RUN_DIR/compose-ps.txt"
  collect_pg_stats baseline

  echo "==> Running k6: script=$LOAD_SCRIPT vus=$VUS duration=$DURATION"
  compose run --rm -v "$ROOT_DIR/$LOAD_SCRIPT:/load-test.js" \
    k6 run --vus "$VUS" --duration "$DURATION" /load-test.js | tee "$RUN_DIR/k6.txt"

  echo "==> Capturing post-run stats"
  collect_pg_stats post
  collect_otel_samples

  echo "==> Stopping profiler stack"
  compose_prof stop profiler otelcol || true

  {
    echo "Run directory: $RUN_DIR"
    echo "Tunables:"
    cat "$RUN_DIR/tunables.env"
    echo
    echo "k6 summary:"
    grep -E 'http_req_duration|http_reqs|checks_failed|checks_succeeded|running \(20|TOTAL RESULTS' "$RUN_DIR/k6.txt" || true
    echo
    echo "otel sample counts:"
    cat "$RUN_DIR/otel-sample-counts.txt"
  } | tee "$RUN_DIR/summary.txt"
}

main() {
  trap restore_pgbouncer_ini EXIT

  require_cmd docker
  require_cmd curl
  require_cmd grep

  case "$COMMAND" in
    reset)
      cleanup_stack
      ;;
    profile)
      run_profile
      ;;
    full)
      cleanup_stack
      run_profile
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"