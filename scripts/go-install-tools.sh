#!/usr/bin/env bash
set -euo pipefail

echo "📦 Installing Go tools from tools.go..."

# Extract the imports from tools.go and install them
for tool in $(go list -f '{{range .Imports}}{{.}} {{end}}' -tags=tools ./tools.go); do
  echo "➡️  Installing $tool"
  go install "$tool@latest"
done

echo "✅ All tools installed!"
