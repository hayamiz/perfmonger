#!/usr/bin/env bash
#
# capture-nvidia-smi-fixtures.sh
#
# Capture real `nvidia-smi` output on a GPU host and turn it into pytest stub
# fixtures for the nvidia-smi GPU integration (ticket #0057). Automates steps
# 3-7 of the capture procedure in
# doc/tickets/0066-nvidia-smi-stub-data-capture.md:
#   3. the canonical --query-gpu invocation (must match #0057),
#   4. capture scenarios (idle / streaming-under-load / with-header),
#   5. an edge-case row exercising [N/A],
#   6. driver/nvidia-smi version stamp,
#   7. the interval-floor probe that feeds #0057's interval-alignment logic.
#
# Fixtures are written per step 8: spec/data/gpu/nvidia-smi_<driver>_<scenario>.csv
#
# ---------------------------------------------------------------------------
# Runbook (the hardware-only steps this script does NOT do):
#
#   1. Provision an ephemeral GPU instance. Default: AWS g4dn.xlarge (T4) with a
#      Deep Learning AMI (driver + CUDA + Docker + PyTorch preinstalled). GCP T4
#      or Lambda Labs are equivalent. Verify with `nvidia-smi`.
#   2. (Optional) This script can generate its own GPU load with --load (needs
#      python3 + torch). Otherwise start a load yourself before running, or run
#      with no load to capture idle-only fixtures.
#   9. TEAR DOWN the instance immediately after capture. Do not leave GPU
#      instances running.
#
# Re-run on a SECOND driver version when possible so the parser is anchored to
# real column/format drift, then hand the interval-probe result back to #0057.
# ---------------------------------------------------------------------------

set -euo pipefail

# --- defaults ---------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTDIR="${REPO_ROOT}/spec/data/gpu"
DURATION=30                 # streaming capture length, seconds
STREAM_INTERVAL_MS=250      # nvidia-smi -lms value for the load-stream fixture
NA_FIELD="fan.speed"        # a field commonly [N/A] on datacenter GPUs (e.g. T4)
GEN_LOAD=0                  # 1 => generate GPU load internally (--load)

# Tier-1 field list -- MUST stay identical to #0057's nvidia-smi invocation.
QUERY='index,name,uuid,utilization.gpu,utilization.memory,memory.used,memory.total,power.draw,power.limit,temperature.gpu'
FMT_RAW='--format=csv,noheader,nounits'
FMT_HDR='--format=csv,nounits'

# --- helpers ----------------------------------------------------------------
log()  { printf '\033[1;32m[capture]\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Usage: ${0##*/} [options]

Capture nvidia-smi fixtures for ticket #0057 (see #0066 for the full procedure).

Options:
  -o, --outdir DIR      Output directory (default: spec/data/gpu)
  -d, --duration SEC    Streaming capture length in seconds (default: ${DURATION})
      --interval-ms MS  -lms value for the load-stream fixture (default: ${STREAM_INTERVAL_MS})
      --na-field FIELD  Extra query field expected to read [N/A] (default: ${NA_FIELD})
      --load            Generate GPU load internally during capture (needs python3+torch)
  -h, --help            Show this help

Run on a GPU host with nvidia-smi available. Tear the host down afterwards.
EOF
}

# --- arg parsing ------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--outdir)      OUTDIR="$2"; shift 2 ;;
    -d|--duration)    DURATION="$2"; shift 2 ;;
    --interval-ms)    STREAM_INTERVAL_MS="$2"; shift 2 ;;
    --na-field)       NA_FIELD="$2"; shift 2 ;;
    --load)           GEN_LOAD=1; shift ;;
    -h|--help)        usage; exit 0 ;;
    *)                die "unknown option: $1 (see --help)" ;;
  esac
done

# --- preflight --------------------------------------------------------------
command -v nvidia-smi >/dev/null 2>&1 || die "nvidia-smi not found on PATH"

DRIVER="$(nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -n1 | tr -d '[:space:]')"
[[ -n "${DRIVER}" ]] || die "could not read driver_version from nvidia-smi"
DRV="${DRIVER//[^0-9A-Za-z.]/_}"   # sanitize for filenames

mkdir -p "${OUTDIR}"
PREFIX="${OUTDIR}/nvidia-smi_${DRV}"
log "driver ${DRIVER} -> fixtures in ${OUTDIR} (prefix nvidia-smi_${DRV})"

# --- optional internal load -------------------------------------------------
LOAD_PID=""
cleanup() {
  if [[ -n "${LOAD_PID}" ]] && kill -0 "${LOAD_PID}" 2>/dev/null; then
    log "stopping GPU load (pid ${LOAD_PID})"
    kill "${LOAD_PID}" 2>/dev/null || true
    wait "${LOAD_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

if [[ "${GEN_LOAD}" -eq 1 ]]; then
  command -v python3 >/dev/null 2>&1 || die "--load needs python3"
  python3 -c 'import torch; assert torch.cuda.is_available()' 2>/dev/null \
    || die "--load needs python3 with CUDA-enabled torch (torch.cuda.is_available() must be True)"
  log "starting internal GPU load (torch matmul)"
  python3 - <<'PY' &
import torch, itertools
a = torch.rand(8192, 8192, device="cuda")
for _ in itertools.repeat(None):
    a = torch.sin(a.mm(a))   # bounded in [-1, 1], keeps the SMs busy
PY
  LOAD_PID=$!
  sleep 5   # let utilization/power ramp before capturing
fi

# --- warn if idle -----------------------------------------------------------
UTIL_NOW="$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits | head -n1 | tr -d '[:space:]')"
if [[ "${UTIL_NOW:-0}" -lt 5 ]] 2>/dev/null; then
  warn "GPU utilization is ${UTIL_NOW}% -- the load-stream fixture will look idle."
  warn "Start a GPU load first, or re-run with --load."
fi

# --- 6. version stamp -------------------------------------------------------
{
  nvidia-smi --version
  echo
  echo "# GPUs:"
  nvidia-smi -L
} > "${PREFIX}_version.txt"
log "wrote $(basename "${PREFIX}_version.txt")"

# --- 4a. idle single snapshot ----------------------------------------------
nvidia-smi --query-gpu="${QUERY}" ${FMT_RAW} > "${PREFIX}_idle.csv"
log "wrote $(basename "${PREFIX}_idle.csv")"

# --- 4c. with-header snapshot (locks header-time index/name/uuid handling) ---
nvidia-smi --query-gpu="${QUERY}" ${FMT_HDR} > "${PREFIX}_with-header.csv"
log "wrote $(basename "${PREFIX}_with-header.csv")"

# --- 5. edge-case row with a likely-[N/A] field -----------------------------
nvidia-smi --query-gpu="${QUERY},${NA_FIELD}" ${FMT_RAW} > "${PREFIX}_na-sample.csv"
if grep -qi '\[N/A\]' "${PREFIX}_na-sample.csv"; then
  log "wrote $(basename "${PREFIX}_na-sample.csv") (contains [N/A])"
else
  warn "$(basename "${PREFIX}_na-sample.csv") has no [N/A]; try a different --na-field"
fi

# --- 4b. streaming under load (primary parsing fixture) ---------------------
log "streaming for ${DURATION}s at -lms ${STREAM_INTERVAL_MS} ..."
timeout "${DURATION}s" \
  nvidia-smi --query-gpu="${QUERY}" ${FMT_RAW} -lms "${STREAM_INTERVAL_MS}" \
  > "${PREFIX}_load-stream.csv" || true   # timeout exits 124; expected
log "wrote $(basename "${PREFIX}_load-stream.csv") ($(wc -l < "${PREFIX}_load-stream.csv") rows)"

# --- 7. interval-floor probe (feeds #0057) ----------------------------------
PROBE_REPORT="${PREFIX}_interval-probe.txt"
: > "${PROBE_REPORT}"
median_delta_ms() {
  # $1 = probe csv whose first column is nvidia-smi's timestamp
  command -v python3 >/dev/null 2>&1 || { echo "n/a (no python3)"; return; }
  python3 - "$1" <<'PY'
import sys, datetime, statistics
ts = []
for line in open(sys.argv[1]):
    line = line.strip()
    if not line:
        continue
    tok = line.split(",")[0].strip()
    for fmt in ("%Y/%m/%d %H:%M:%S.%f", "%Y/%m/%d %H:%M:%S"):
        try:
            ts.append(datetime.datetime.strptime(tok, fmt))
            break
        except ValueError:
            continue
deltas = [(ts[i] - ts[i-1]).total_seconds() * 1000 for i in range(1, len(ts))]
print(f"{statistics.median(deltas):.1f}" if deltas else "n/a (too few samples)")
PY
}

log "probing interval floor (this drives #0057's -i alignment check) ..."
{
  echo "# nvidia-smi interval-floor probe -- driver ${DRIVER}"
  echo "# requested_interval    median_actual_delta_ms"
} >> "${PROBE_REPORT}"
for MS in 50 100 250 500; do
  f="${PREFIX}_probe-${MS}ms.csv"
  timeout 5s nvidia-smi --query-gpu=timestamp,utilization.gpu \
    ${FMT_RAW} -lms "${MS}" > "${f}" || true
  printf '%-22s %s\n' "-lms ${MS}" "$(median_delta_ms "${f}")" >> "${PROBE_REPORT}"
done
f="${PREFIX}_probe-1s.csv"
timeout 5s nvidia-smi --query-gpu=timestamp,utilization.gpu \
  ${FMT_RAW} -l 1 > "${f}" || true
printf '%-22s %s\n' "-l 1" "$(median_delta_ms "${f}")" >> "${PROBE_REPORT}"

log "wrote $(basename "${PROBE_REPORT}"):"
cat "${PROBE_REPORT}" >&2

# --- summary ----------------------------------------------------------------
log "done. Fixtures for driver ${DRIVER}:"
ls -1 "${PREFIX}"* >&2
cat >&2 <<EOF

Next:
  * Report the interval-probe result to #0057: the smallest requested interval
    where median_actual_delta_ms is close to it is the practical floor. If -lms
    sub-second values clamp to ~1000ms, perfmonger's -i 0.1 must be rejected.
  * Re-run on a second driver version if available.
  * Tear down this GPU instance now (runbook step 9).
EOF
