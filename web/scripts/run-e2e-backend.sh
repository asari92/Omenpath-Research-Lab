#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
runtime_dir="$(mktemp -d /tmp/omenpath-e2e.XXXXXX)"
database_path="$runtime_dir/lab.sqlite"
server_pid=""

cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -f "$database_path" "$database_path-shm" "$database_path-wal"
  rmdir "$runtime_dir" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

cd "$repo_root"
OMENPATH_ADDR=127.0.0.1:18080 OMENPATH_DB_PATH="$database_path" \
  go run ./cmd/server &
server_pid="$!"
wait "$server_pid"
