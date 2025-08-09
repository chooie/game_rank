#!/usr/bin/env bash

set -euo pipefail

# Get the directory containing this script
SCRIPT_DIR="$(dirname "$0")"

# Load environment variables from .env in the same directory
if [ -f "$SCRIPT_DIR/.env" ]; then
  export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

# Default NODE_ENV if not already set
NODE_ENV="${NODE_ENV:-development}"

SRC_DIR="$SCRIPT_DIR/../src"

if [ "$NODE_ENV" = "development" ]; then
  echo "🚀 Starting Express server in development mode..."
  npx nodemon --watch "$SRC_DIR" --exec "node $SRC_DIR/server.js"
else
  echo "🚀 Starting Express server in production mode..."
  node "$SRC_DIR/server.js"
fi
