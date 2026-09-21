#!/usr/bin/env bash
# Runs the whole Summit dev stack locally and restarts the Go services when
# their sources change:
#
#   summit       game server   auth WS ws://127.0.0.1:5001/realm/auth, realm WS :5002/realms/<slug>, TCP :5000/:8129
#   assetserver  asset server  http://localhost:8080 (JIT converts from MPQs / raw files)
#   client       Vite dev server http://localhost:5173, wired to the two above
#
# Usage:
#   scripts/dev.sh                 # start everything, Ctrl-C stops everything
#   WOW_DATA=~/wow/Data scripts/dev.sh   # let the asset server read your MPQs
#
# Environment:
#   WOW_DATA        WoW client Data/ directory with the MPQs (optional)
#   ASSET_UPSTREAM  asset server to fetch missing source files from when there
#                   are no local MPQs, e.g. https://assets-summit.dev.pilab.hu
#   ASSET_DIR       raw asset directory for the asset server (default: client/assets)
#   ASSET_CACHE     conversion cache (default: .dev/asset-cache)
#   ASSET_PORT      default 8080        CLIENT_PORT   default 5173
#   WATCH_INTERVAL  seconds between source scans when fswatch is missing (default 1)
#   NO_CLIENT=1     skip the Vite dev server
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$PWD"

ASSET_DIR="${ASSET_DIR:-$ROOT/client/assets}"
ASSET_CACHE="${ASSET_CACHE:-$ROOT/.dev/asset-cache}"
ASSET_PORT="${ASSET_PORT:-8080}"
CLIENT_PORT="${CLIENT_PORT:-5173}"
WATCH_INTERVAL="${WATCH_INTERVAL:-1}"
BIN="$ROOT/.dev/bin"

# Go source roots the watcher tracks; a change rebuilds and restarts both services
WATCH_PATHS=(cmd pkg internal go.mod go.sum)

mkdir -p "$BIN" "$ASSET_CACHE" "$ASSET_DIR" "$ROOT/.dev"

# ---------------------------------------------------------------- output ----
c_summit=$'\033[1;35m'; c_asset=$'\033[1;36m'; c_client=$'\033[1;33m'; c_dev=$'\033[1;32m'; c_off=$'\033[0m'

log() { printf '%s[dev]%s %s\n' "$c_dev" "$c_off" "$*"; }

# prefix <tag> <color>: pipe a service's output through with a colored tag
prefix() {
  local tag="$1" color="$2"
  while IFS= read -r line; do
    printf '%s[%s]%s %s\n' "$color" "$tag" "$c_off" "$line"
  done
}

# --------------------------------------------------------------- services ----
summit_pid=""
asset_pid=""
client_pid=""

build_go() {
  log "building Go services"
  go build -o "$BIN/summit" ./cmd/summit &&
    go build -o "$BIN/assetserver" ./cmd/assetserver
}

start_summit() {
  # The server reads summit.yaml / summit-store.yaml from its working directory
  (cd "$ROOT" && exec "$BIN/summit") > >(prefix summit "$c_summit") 2>&1 &
  summit_pid=$!
  log "summit started (pid $summit_pid)"
}

start_assetserver() {
  local args=(--listen ":$ASSET_PORT" --assets "$ASSET_DIR" --cache "$ASSET_CACHE")
  if [[ -n "${WOW_DATA:-}" ]]; then
    args+=(--mpq "$WOW_DATA")
  fi
  if [[ -n "${ASSET_UPSTREAM:-}" ]]; then
    args+=(--upstream "$ASSET_UPSTREAM")
  fi
  "$BIN/assetserver" "${args[@]}" > >(prefix asset "$c_asset") 2>&1 &
  asset_pid=$!
  log "assetserver started (pid $asset_pid) on :$ASSET_PORT"
}

start_client() {
  [[ "${NO_CLIENT:-}" == 1 ]] && return 0
  if [[ ! -d client/node_modules ]]; then
    log "installing client dependencies"
    (cd client && npm install)
  fi
  (
    cd client &&
      VITE_ASSET_HOST="${VITE_ASSET_HOST:-http://localhost:$ASSET_PORT}" \
      VITE_AUTH_WS_URL="ws://127.0.0.1:5001/realm/auth" \
      exec npm run dev -- --port "$CLIENT_PORT" --strictPort
  ) > >(prefix client "$c_client") 2>&1 &
  client_pid=$!
  log "client dev server starting on http://localhost:$CLIENT_PORT"
}

# kill_tree <pid>: terminates a process and all of its descendants
kill_tree() {
  local pid="$1" child
  [[ -n "$pid" ]] || return 0
  for child in $(pgrep -P "$pid" 2>/dev/null); do
    kill_tree "$child"
  done
  kill "$pid" 2>/dev/null || true
}

stop_pid() {
  local pid="$1"
  [[ -n "$pid" ]] || return 0
  kill -0 "$pid" 2>/dev/null || return 0
  kill_tree "$pid"
  for _ in $(seq 1 30); do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 0.1
  done
  kill -9 "$pid" 2>/dev/null || true
}

restart_go_services() {
  stop_pid "$summit_pid"; summit_pid=""
  stop_pid "$asset_pid"; asset_pid=""
  if build_go; then
    start_summit
    start_assetserver
  else
    log "build failed; services stay down until the next successful build"
  fi
}

watch_pid=""

cleanup() {
  trap - INT TERM EXIT
  log "shutting down"
  stop_pid "$watch_pid"
  stop_pid "$summit_pid"
  stop_pid "$asset_pid"
  stop_pid "$client_pid"
  # Safety net for anything that escaped the tree
  pkill -f "$BIN/summit" 2>/dev/null || true
  pkill -f "$BIN/assetserver" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

# ---------------------------------------------------------------- watcher ----
# Emits one line per change batch on stdout.
watch_sources() {
  if command -v fswatch >/dev/null 2>&1; then
    fswatch -o -l 0.3 --include='\.go$' --include='go\.(mod|sum)$' --exclude='.*' "${WATCH_PATHS[@]}"
  else
    # Portable fallback: poll for files newer than a stamp file
    local stamp="$ROOT/.dev/.watch-stamp"
    touch "$stamp"
    while true; do
      sleep "$WATCH_INTERVAL"
      if [[ -n "$(find "${WATCH_PATHS[@]}" \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer "$stamp" -print -quit 2>/dev/null)" ]]; then
        touch "$stamp"
        echo change
      fi
    done
  fi
}

# ------------------------------------------------------------------- main ----
log "summit dev stack (Ctrl-C to stop)"
[[ -n "${WOW_DATA:-}" ]] && log "MPQs: $WOW_DATA" || log "no WOW_DATA set: asset server serves $ASSET_DIR only"
[[ -n "${ASSET_UPSTREAM:-}" ]] && log "upstream assets: $ASSET_UPSTREAM"

build_go
start_summit
start_assetserver
start_client

command -v fswatch >/dev/null 2>&1 && log "watching Go sources with fswatch" || log "watching Go sources (poll every ${WATCH_INTERVAL}s; brew install fswatch for instant restarts)"

watch_loop() {
  watch_sources | while IFS= read -r _; do
    # Coalesce bursts of saves
    sleep 0.5
    log "source change detected, rebuilding"
    restart_go_services
  done
}

# The watcher runs in the background so the main shell sits in an interruptible
# wait: Ctrl-C, or a kill from elsewhere, reaches the cleanup trap immediately.
watch_loop &
watch_pid=$!
wait "$watch_pid"
