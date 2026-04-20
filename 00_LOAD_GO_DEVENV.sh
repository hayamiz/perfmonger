#!/usr/bin/env bash

# Go development environment loader for this repo.
# Usage: source 00_LOAD_GO_DEVENV.sh

# Ensure this script is sourced, not executed
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo "Please source this script: source 00_LOAD_GO_DEVENV.sh" >&2
  exit 1
fi

# Preserve prior env to allow easy unload
export PERF_GOENV_PREV_PATH="${PATH}"
export PERF_GOENV_PREV_GOROOT="${GOROOT:-}"
export PERF_GOENV_PREV_GOPATH="${GOPATH:-}"
export PERF_GOENV_PREV_GOBIN="${GOBIN:-}"
export PERF_GOENV_PREV_GOCACHE="${GOCACHE:-}"
export PERF_GOENV_PREV_GOMODCACHE="${GOMODCACHE:-}"
export PERF_GOENV_PREV_GO111MODULE="${GO111MODULE:-}"
export PERF_GOENV_PREV_GOPROXY="${GOPROXY:-}"

# Repo root
_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PERF_REPO_ROOT="${_script_dir}"

# Local dev env root
export GOENV_ROOT="${PERF_REPO_ROOT}/godevenv"
export GOPATH="${GOENV_ROOT}/gopath"
export GOBIN="${GOPATH}/bin"
export GOCACHE="${GOENV_ROOT}/cache"
export GOMODCACHE="${GOPATH}/pkg/mod"
export GO111MODULE=on
# Default to direct; adjust to your proxy/mirror or set GOPROXY before sourcing
export GOPROXY="${GOPROXY:-direct}"

# Create directories
mkdir -p "${GOBIN}" "${GOCACHE}" "${GOMODCACHE}"

# Detect go and set a sane GOROOT
_go_bin="$(command -v go 2>/dev/null || true)"
if [[ -n "${_go_bin}" ]]; then
  # Prefer detected go's GOROOT; override any incorrect GOROOT in the shell
  export GOROOT="$(${_go_bin} env GOROOT 2>/dev/null)"
else
  # No system go found. Optionally, place a Go toolchain under ${GOENV_ROOT}/go and set GOROOT accordingly.
  # You can install it manually (e.g., extract Go to ${GOENV_ROOT}/go) and uncomment:
  # export GOROOT="${GOENV_ROOT}/go"
  echo "[WARN] 'go' not found on PATH. Install Go or place a toolchain under \"${GOENV_ROOT}/go\" and set GOROOT accordingly." >&2
fi

# Update PATH
if [[ -n "${GOROOT:-}" ]]; then
  export PATH="${GOBIN}:${GOROOT}/bin:${PATH}"
else
  export PATH="${GOBIN}:${PATH}"
fi

# Helper: print info
go_devenv_info() {
  echo "== Go DevEnv =="
  echo "REPO        : ${PERF_REPO_ROOT}"
  echo "GOROOT      : ${GOROOT:-<unset>}"
  echo "GOPATH      : ${GOPATH}"
  echo "GOBIN       : ${GOBIN}"
  echo "GOCACHE     : ${GOCACHE}"
  echo "GOMODCACHE  : ${GOMODCACHE}"
  echo "GO111MODULE : ${GO111MODULE}"
  echo "GOPROXY     : ${GOPROXY}"
  if command -v go >/dev/null 2>&1; then
    echo -n "go version : "; go version || true
  else
    echo "go version : <not installed>"
  fi
}

# Helper: unload and restore prior env
go_devenv_unload() {
  export PATH="${PERF_GOENV_PREV_PATH}"
  export GOROOT="${PERF_GOENV_PREV_GOROOT}"
  export GOPATH="${PERF_GOENV_PREV_GOPATH}"
  export GOBIN="${PERF_GOENV_PREV_GOBIN}"
  export GOCACHE="${PERF_GOENV_PREV_GOCACHE}"
  export GOMODCACHE="${PERF_GOENV_PREV_GOMODCACHE}"
  export GO111MODULE="${PERF_GOENV_PREV_GO111MODULE}"
  export GOPROXY="${PERF_GOENV_PREV_GOPROXY}"
  unset PERF_GOENV_PREV_PATH PERF_GOENV_PREV_GOROOT PERF_GOENV_PREV_GOPATH PERF_GOENV_PREV_GOBIN \
        PERF_GOENV_PREV_GOCACHE PERF_GOENV_PREV_GOMODCACHE PERF_GOENV_PREV_GO111MODULE PERF_GOENV_PREV_GOPROXY \
        PERF_REPO_ROOT GOENV_ROOT
  echo "Go DevEnv unloaded."
}

echo "Go DevEnv loaded. Run 'go_devenv_info' to inspect, 'go_devenv_unload' to restore."

