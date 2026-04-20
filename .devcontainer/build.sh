#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

export UID
export GID="$(id -g)"
GH_CONFIG_DIR="${GH_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/gh}"
if [ ! -d "$GH_CONFIG_DIR" ]; then
  GH_CONFIG_DIR="$(mktemp -d)"
  echo "Note: gh config dir not found, using empty temp dir: $GH_CONFIG_DIR"
fi
export GH_CONFIG_DIR

echo "Building devcontainer (UID=$UID, GID=$GID)..."
devcontainer build --workspace-folder "$PROJECT_DIR"
echo "Build complete."
