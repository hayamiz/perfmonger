---
title: Capture real nvidia-smi output to generate pytest stub fixtures
type: test
priority: medium
status: open
created: 2026-08-05
updated: 2026-08-08
---


## Description

Split out from [#0057](0057-gpu-usage-external-tool-integration.md) (nvidia-smi
GPU integration). That ticket's testing decision is **pytest + a mock
`nvidia-smi` stub**: a stub executable on `PATH` emits canned CSV so the parsing,
latest-value pairing, stale-flag, and enablement/failure behavior can be verified
on the GPU-less CI host.

For that stub to be faithful, its canned CSV must come from **real hardware**,
not hand-authored guesses. This ticket owns the **procedure for capturing that
data** on an ephemeral cloud GPU instance (the Tier 2 environment chosen in
#0057) and turning it into committed test fixtures.

This is deliberately separate from #0057 because it requires GPU hardware access
and a repeatable capture runbook, whereas #0057's implementation and Tier 1
tests run GPU-less.

## Scope

Capture, on a real NVIDIA GPU box, the outputs that the #0057 parser must handle,
and commit them as fixtures plus a documented capture procedure.

### What to capture

- **The exact `--query-gpu` invocation #0057 will use**, in
  `--format=csv,noheader,nounits` form, covering the Tier 1 fields:
  `index,name,uuid,utilization.gpu,utilization.memory,memory.used,memory.total,power.draw,power.limit,temperature.gpu`.
- **A short streaming capture** (`-l <interval>`) under a synthetic GPU load so
  utilization/power move across samples (not a single idle snapshot).
- **Edge-case rows** where available: `[N/A]` values (fields the driver/GPU does
  not report), and — if the instance type allows — **multi-GPU** and **MIG**
  output so column/row handling is exercised.
- **Ideally two driver versions** so the parser's tolerance to column/format
  drift is anchored to real variance.
- **nvidia-smi's minimum supported `-l` interval** and its behavior when asked
  for a finer interval — this value feeds #0057's interval-alignment / startup
  failure logic.

### Deliverables

1. Fixture CSV files committed under the test-data location (e.g. `spec/data/`
   or `tests/stubs/`), named so #0057's pytest can load them.
2. A documented, repeatable **capture procedure / runbook** (a short script under
   `tools/` and/or notes) so fixtures can be refreshed on a driver bump. This can
   share the ephemeral-cloud-GPU setup with #0057's `tools/gpu-smoke.sh`.
3. A recorded note of nvidia-smi's minimum `-l` interval, handed back to #0057.

## Procedure

Concrete, repeatable runbook. Steps 1–2 provision, 3–6 capture, 7 measures the
interval floor for #0057, 8 lays out fixtures, 9 tears down. A committed script
(step 10) automates 3–7.

### 1. Provision an ephemeral GPU instance

- Default: **AWS `g4dn.xlarge`** (1x NVIDIA T4) with the **Deep Learning AMI
  (Ubuntu)** — driver, CUDA toolkit (`nvcc`), Docker, and PyTorch come
  preinstalled, so no driver install is needed. GCP `n1` + T4 or Lambda Labs are
  equivalent alternatives.
- Confirm the tooling is live before capturing:
  ```bash
  nvidia-smi                       # driver + GPU visible
  nvidia-smi --version             # record NVIDIA-SMI + driver version for filenames
  ```
- For **multi-GPU / MIG** fixtures (optional but valuable): use a 2+ GPU instance
  (e.g. `g4dn.12xlarge`) and/or enable MIG on an A100/A30 instance
  (`nvidia-smi -mig 1` + create instances). If unavailable, skip and note it —
  single-GPU fixtures are the must-have.

### 2. Prepare a GPU load

Fixtures must show *moving* utilization/power, not an idle snapshot. Pick one:

- **Simple (DLAMI has PyTorch):**
  ```bash
  python3 -c "import torch,itertools; a=torch.rand(8192,8192,device='cuda'); \
    [a.mm(a) for _ in itertools.repeat(None, 200000)]" &   # Ctrl-C / kill when done
  ```
- **Dedicated (gpu-burn):**
  ```bash
  git clone https://github.com/wilicc/gpu-burn && (cd gpu-burn && make && ./gpu_burn 120) &
  ```

### 3. The canonical query (must match #0057's invocation)

```bash
QUERY='index,name,uuid,utilization.gpu,utilization.memory,\
memory.used,memory.total,power.draw,power.limit,temperature.gpu'
FMT='--format=csv,noheader,nounits'
```

### 4. Capture scenarios

```bash
# a) idle single snapshot (baseline, some fields may read [N/A])
nvidia-smi --query-gpu="$QUERY" $FMT            > idle.csv

# b) streaming under load (~30s) — the primary parsing fixture
nvidia-smi --query-gpu="$QUERY" $FMT -lms 250   > load-stream.csv   # Ctrl-C after ~30s

# c) with header, to lock the header-time index/name/uuid handling
nvidia-smi --query-gpu="$QUERY" --format=csv,nounits > with-header.csv
```

### 5. Edge-case rows

- Capture at least one row exhibiting `[N/A]` (fields the T4/driver does not
  report — e.g. some power/clock fields on certain SKUs). If none appear
  naturally, query an extra field known to be unsupported to produce a genuine
  `[N/A]` sample the parser must tolerate.
- If multi-GPU/MIG is available, capture the multi-row / MIG-partition output as
  `multi-gpu.csv` / `mig.csv`.

### 6. Record versions

Save `nvidia-smi --version` output alongside the CSVs; **ideally repeat the whole
capture on a second driver version** so the parser is anchored to real
column/format drift.

### 7. Interval-floor probe (feeds #0057's alignment / startup-failure logic)

This is the key hardware-only finding. `perfmonger record` supports sub-second
intervals (`-i 0.1` = 100 ms), but `nvidia-smi`'s own sampling cadence has a
floor set by driver query latency. Measure the *actual* achievable interval:

```bash
# request several intervals; the `timestamp` column reveals the real delta
for MS in 50 100 250 500; do
  echo "== requested ${MS}ms =="
  timeout 5 nvidia-smi --query-gpu=timestamp,utilization.gpu \
    --format=csv,noheader,nounits -lms "$MS" > "probe_${MS}ms.csv"
done
timeout 5 nvidia-smi --query-gpu=timestamp,utilization.gpu \
  --format=csv,noheader,nounits -l 1 > probe_1s.csv
```

For each file compute the median delta between successive `timestamp` values.
Report to #0057: the smallest requested interval where actual ≈ requested (the
practical floor), and confirm whether `-lms` sub-second is honored or clamps to
~1 s on this driver. **This value is what #0057's "fail record startup if `-i`
is finer than nvidia-smi can honor" check compares against.**

### 8. Fixture layout & naming

- Captured CSVs → **`spec/data/gpu/`** (resolvable by the existing `data_file`
  fixture, e.g. `data_file("gpu/nvidia-smi_550.90_load-stream.csv")`).
- The mock stub executable (built in #0057) → **`tests/stubs/nvidia-smi`**; #0057's
  pytest prepends `tests/stubs/` to `PATH` (note: `run_perfmonger` in
  `tests/conftest.py` does not set `env` today — #0057 adds an env/PATH override).
- Naming: `nvidia-smi_<driver>_<scenario>.csv`, e.g.
  `nvidia-smi_550.90_load-stream.csv`, `..._idle.csv`, `..._multi-gpu.csv`.

### 9. Teardown

Terminate/stop the cloud instance immediately after capture. Total live time is
minutes; do not leave GPU instances running.

### 10. Deliverable script

Commit `tools/capture-nvidia-smi-fixtures.sh` implementing steps 3–7 (query,
scenarios, edge cases, version stamp, interval probe → files named per step 8)
plus a short runbook covering steps 1–2 and 9. Reuse the ephemeral-cloud-GPU
setup shared with #0057's `tools/gpu-smoke.sh`.

## Notes

- Blocks the mock-stub pytest work in #0057 from being fully faithful, but not
  #0057's implementation itself — #0057 can start against a provisional
  hand-authored stub and swap in the captured fixtures once this lands.
- Reuse the same ephemeral cloud GPU instance as the #0057 Tier 2 smoke test to
  avoid provisioning twice.
