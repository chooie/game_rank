#!/usr/bin/env bash

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Resolve directories
# ──────────────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(dirname "$0")"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
DIST_CSS_DIR="$ROOT_DIR/public/dist/css"

SERVER_ENTRY="$SRC_DIR/server.js"

# ──────────────────────────────────────────────────────────────────────────────
# Load .env (preserve quoted values and spaces)
# ──────────────────────────────────────────────────────────────────────────────
if [[ -f "$SCRIPT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1090
  . "$SCRIPT_DIR/.env"
  set +a
fi

# Default NODE_ENV if not already set
NODE_ENV="${NODE_ENV:-development}"

# ──────────────────────────────────────────────────────────────────────────────
# Binaries
# ──────────────────────────────────────────────────────────────────────────────
NODE_BIN="${NODE_BIN:-node}"
NPM_BIN="${NPM_BIN:-npm}"

NODEMON_BIN="$ROOT_DIR/node_modules/.bin/nodemon"
SASS_BIN="$ROOT_DIR/node_modules/.bin/sass"

ensure_dir() {
  mkdir -p "$1"
}

run_dev_server() {
  echo "🚀 Starting Express server in development mode..."
  if command -v "$NODEMON_BIN" >/dev/null 2>&1; then
    "$NODEMON_BIN" --watch "$SRC_DIR" --ext js,json,hbs,handlebars,scss --exec "$NODE_BIN $SERVER_ENTRY"
  else
    echo "⚠️  nodemon not found; starting plain node (no auto-reload)."
    "$NODE_BIN" "$SERVER_ENTRY"
  fi
}

run_prod_server() {
  echo "🚀 Starting Express server in production mode..."
  "$NODE_BIN" "$SERVER_ENTRY"
}

build_scss_dev() {
  if ! command -v "$SASS_BIN" >/dev/null 2>&1; then
    echo "ℹ️  sass not found; skipping SCSS watch. Run: $NPM_BIN i -D sass"
    return
  fi
  echo "🎨 Watching SCSS (dev, source maps, expanded)…"
  ensure_dir "$DIST_CSS_DIR"

  # Compile *every non-partial* SCSS in templates → individual CSS files
  # This fits your “one CSS per page .handlebars” approach.
  "$SASS_BIN" \
    --watch "$SRC_DIR/templates:$DIST_CSS_DIR" \
    --style=expanded \
    --embed-source-map \
    --quiet \
    &
  pids+=("$!")
}

build_scss_prod() {
  if ! command -v "$SASS_BIN" >/dev/null 2>&1; then
    echo "ℹ️  sass not found; skipping SCSS build. Run: $NPM_BIN i -D sass"
    return
  fi
  echo "🎨 Building SCSS (prod, compressed)…"
  ensure_dir "$DIST_CSS_DIR"

  # One-shot build, minified. Underscored files (_*.scss) are skipped automatically.
  "$SASS_BIN" \
    "$SRC_DIR/templates:$DIST_CSS_DIR" \
    --style=compressed \
    --no-source-map \
    --quiet
}

# ──────────────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────────────
if [[ "$NODE_ENV" == "development" ]]; then
  build_scss_dev
  run_dev_server
else
  build_scss_prod
  run_prod_server
fi
