#!/usr/bin/env bash
# initialize.sh — Runs on the HOST before the devcontainer starts.
# Ensures required files/directories exist so bind mounts don't fail.
set -euo pipefail

# Ensure a valid SSH agent socket exists for devcontainer bind mount.
# Self-contained: does not depend on .shellcore.sh or other dotfiles.
_ensure_ssh_agent() {
    local agent_file="$HOME/$(hostname -s)/.ssh/ssh-agent"

    # Try to restore from saved agent file
    if [ -f "$agent_file" ]; then
        eval "$(cat "$agent_file")" >/dev/null 2>&1 || true
    fi

    # If socket is missing or dead, start a new agent
    if [ -z "${SSH_AUTH_SOCK:-}" ] || [ ! -S "$SSH_AUTH_SOCK" ]; then
        mkdir -p "$(dirname "$agent_file")"
        ssh-agent -s > "$agent_file"
        eval "$(cat "$agent_file")" >/dev/null 2>&1
    fi

    # Update stable symlink for devcontainer.json bind mount
    ln -sfn "$SSH_AUTH_SOCK" "$HOME/.ssh-auth-sock"
}
_ensure_ssh_agent

# Ensure gh CLI config directory exists for bind mount
if [ -z "${GH_CONFIG_DIR:-}" ]; then
  export GH_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/gh"
fi
if [ ! -d "$GH_CONFIG_DIR" ]; then
  mkdir -p "$GH_CONFIG_DIR"
fi

# Stage OpenCode config for bind mount.
# The source directory may contain symlinks (e.g. from dotopencode/setup.sh)
# that point to paths not available inside the container. We copy with -L to
# dereference them so the mount contains plain files the container can read.
OPENCODE_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode"
OPENCODE_STAGE="$HOME/.devcontainer-opencode-config"
mkdir -p "$OPENCODE_CONFIG_DIR"
mkdir -p "$OPENCODE_STAGE"
find "$OPENCODE_STAGE" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
if [ -n "$(ls -A "$OPENCODE_CONFIG_DIR" 2>/dev/null)" ]; then
  cp -rL "$OPENCODE_CONFIG_DIR/." "$OPENCODE_STAGE/"
fi

# Stage Claude config for bind mount.
# ~/.claude on the host typically contains symlinks into a dotfiles repo that
# the container cannot resolve. Copy with -L to dereference symlinks so the
# mount exposes plain files. Whitelist entries to avoid copying large runtime
# directories (projects/, sessions/, statsig/, ...).
CLAUDE_SRC="${HOME}/.claude"
CLAUDE_STAGE="${HOME}/.devcontainer-claude-stage"
mkdir -p "$CLAUDE_STAGE"
chmod 700 "$CLAUDE_STAGE"
find "$CLAUDE_STAGE" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
for _item in settings.json settings.local.json CLAUDE.md \
             notification_wrapper.sh stop_wrapper.sh osc9_notify.sh skills .credentials.json; do
  if [ -e "$CLAUDE_SRC/$_item" ]; then
    cp -rL "$CLAUDE_SRC/$_item" "$CLAUDE_STAGE/"
  fi
done
unset _item

# Ensure Claude state file exists for bind mount.
mkdir -p "${HOME}/.claude"
touch -a "${HOME}/.claude.json"

# Ensure persistent Claude Code data directories exist for bind mounts.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "${SCRIPT_DIR}/claude-projects"
mkdir -p "${SCRIPT_DIR}/claude-sessions"
mkdir -p "${SCRIPT_DIR}/claude-plugins"
mkdir -p "${SCRIPT_DIR}/claude-file-history"

# Ensure pushover token file exists (empty is fine) so bind mount doesn't fail
touch -a "${HOME}/.pushover_token.sh"
