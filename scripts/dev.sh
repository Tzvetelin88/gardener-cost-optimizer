#!/bin/sh

set -eu

API_ADDR="${API_ADDR:-:8080}"
DATA_SOURCE="${DATA_SOURCE:-mock}"
FRONTEND_PORT="${FRONTEND_PORT:-4173}"
VITE_API_BASE_URL="${VITE_API_BASE_URL:-http://localhost:8080/api/v1}"

case "$DATA_SOURCE" in
  mock|real|auto) ;;
  *)
    echo "Unsupported DATA_SOURCE: $DATA_SOURCE"
    echo "Use one of: mock, real, auto"
    exit 1
    ;;
esac

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"

echo "Installing frontend dependencies..."
(cd "$FRONTEND_DIR" && npm install)

cleanup() {
  if [ "${BACKEND_PID:-}" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi

  if [ "${FRONTEND_PID:-}" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed or not on PATH. Frontend started, but backend was not launched."
  echo "Install Go and run: cd \"$BACKEND_DIR\" && go run ./cmd/api"
  echo "Frontend PID: $FRONTEND_PID"
  wait "$FRONTEND_PID"
  exit 0
fi

(cd "$BACKEND_DIR" && API_ADDR="$API_ADDR" DATA_SOURCE="$DATA_SOURCE" go run ./cmd/api) &
BACKEND_PID=$!

echo "Building frontend bundle..."
(cd "$FRONTEND_DIR" && VITE_API_BASE_URL="$VITE_API_BASE_URL" npm run build)

echo "Starting Smart Cost Optimizer frontend preview..."
(cd "$FRONTEND_DIR" && npm run preview -- --host=0.0.0.0 --port="$FRONTEND_PORT") &
FRONTEND_PID=$!

echo "Frontend PID: $FRONTEND_PID"
echo "Backend PID: $BACKEND_PID"
echo "Backend DATA_SOURCE: $DATA_SOURCE"
echo "Frontend URL: http://localhost:$FRONTEND_PORT/"
echo "Press Ctrl+C to stop both processes."

wait "$FRONTEND_PID" "$BACKEND_PID"
