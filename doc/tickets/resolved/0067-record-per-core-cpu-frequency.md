---
title: Record per-core CPU operating frequency with minimal overhead (auto-detected)
type: feature
priority: high
status: resolved
created: 2026-11-20
updated: 2026-11-20
---

## Description

The `20261112_ICDE_DEFT` experiments study CPU DVFS behaviour, and the
recordings currently contain only `/proc/stat`-derived utilisation. We want
`record` to also capture the **current operating frequency of every CPU
core** at each sample, while adding as little load to the measured system as
possible (in particular: no IPIs that wake idle cores and perturb C-state /
DVFS behaviour, no root requirement, no extra processes).

Requested behaviour (from the user): at command start-up, check whether the
frequency information can be obtained (cpufreq sysfs etc.). If it cannot,
silently disable frequency recording and record only the existing CPU
information; if it can, record the frequencies in addition.

## Investigation

### Candidate sources

| Source | Privilege | Cost per sample | Verdict |
| --- | --- | --- | --- |
| `/sys/devices/system/cpu/cpuN/cpufreq/scaling_cur_freq` | none | 1 `pread` per core on a kept-open fd | **chosen** (see kernel caveat) |
| `cpuinfo_cur_freq` | root (0400) | calls driver `->get()`, e.g. MSR read via IPI on acpi-cpufreq | rejected |
| `/proc/cpuinfo` "cpu MHz" | none | ~20 KB text for 16 CPUs (≈1.5 KB/CPU), 280 µs/read here; same kernel path as `scaling_cur_freq` | rejected (heavier, same data) |
| APERF/MPERF via `/dev/cpu/N/msr` (turbostat style) | root + `msr` module | IPI (or migration) to every core per sample | rejected (wakes idle cores) |
| perf_event `msr` PMU (aperf/mperf) | CAP_PERFMON / paranoid ≤ 0 | read of a CPU-bound event IPIs the target CPU | rejected |
| `power:cpu_frequency` tracepoint / eBPF | root | event-driven, but not emitted under HWP (hardware-controlled P-states) | rejected |

### Kernel behaviour of `scaling_cur_freq` (x86)

`show_scaling_cur_freq()` first calls `arch_freq_get_on_cpu(policy->cpu)`,
then falls back to `cpufreq_driver->get()` (only for `setpolicy` drivers) or
`policy->cur` (verified in v5.15 and v6.8 `drivers/cpufreq/cpufreq.c`).

- **v5.19 and later** (`arch/x86/kernel/cpu/aperfmperf.c`): tick-based. The
  scheduler tick (`arch_scale_freq_tick()`) caches the APERF/MPERF deltas
  per CPU; `arch_freq_get_on_cpu()` only reads that cache under a seqcount.
  If the sample is older than `MAX_SAMPLE_AGE` (`HZ/50` = 20 ms, i.e. idle /
  NOHZ-full CPU), it falls back to `cpufreq_quick_get()` (last requested
  frequency). **No IPI, no sleep.**
- **v5.18 and earlier** (checked v5.15, v5.17, v5.18): on-demand. Each read
  whose cached sample is ≥ 10 ms old does a synchronous
  `smp_call_function_single()` to the target CPU (idle CPUs are *not*
  skipped), and if the sample is > 1000 ms old it additionally
  `msleep(10)`s and IPIs again. With a ≥ 1 s recording interval this means
  one or two IPIs per core per sample, and up to N × 10 ms of reader latency
  per sweep. This is unacceptable for DVFS measurements, so frequency
  recording must be **disabled on x86 kernels < 5.19**. (Distro kernels may
  have backported the rework; the version gate errs on the safe side.)

On non-x86 architectures the generic `arch_freq_get_on_cpu()` returns 0 and
the value comes from `policy->cur` (cheap, no IPI).

### Measurements on the dev host

Xeon E5-2680 (16 CPUs), kernel 6.8.0, `intel_cpufreq` (intel_pstate passive)
+ `schedutil`, one cpufreq policy per CPU:

- Hammering all 16 `scaling_cur_freq` fds with `pread(fd, 32, 0)` for 5 s
  (~2.7 M reads) produced **no increase** in `/proc/interrupts` `CAL`
  (function-call IPIs) versus an idle 5 s baseline (2093 vs 3225).
- ~1.8 µs per `pread` → ~30 µs per 16-core sweep. Re-opening files on each
  sample costs ~30× more (≈ 900 µs/sweep from Python), so fds must be kept
  open and re-read at offset 0.

### Semantics to document

- On x86 ≥ 5.19 the value is the APERF/MPERF-derived frequency over the
  **last scheduler tick before the read** (a few ms), not an average over the
  recording interval.
- For a core idle for > 20 ms the value is the cpufreq fallback (last
  requested / policy frequency), not a measured one.
- When several CPUs share one cpufreq policy, all of them report the policy
  CPU's frequency (correct for a shared clock domain).

## Implementation Notes

- **Collector** (`core/internal/perfmonger`): a `CpuFreqReader` that, at
  start-up, opens `cpuN/cpufreq/scaling_cur_freq` for every core and keeps
  the fds open; each sample does one `pread(fd, buf, 0)` per core and parses
  the kHz integer. The sysfs root is injectable for tests.
- **Start-up probe** (in order):
  1. `--no-cpufreq` given → disabled.
  2. x86 (`amd64`/`386`) and kernel release < 5.19 → disabled, with a
     one-line notice on stderr (cpufreq exists but is deliberately not
     read).
  3. No core's `scaling_cur_freq` can be opened and read → disabled silently
     (VMs/containers without cpufreq); reason shown only in debug mode
     (`PERFMONGER_DEBUG`).
  4. Otherwise enabled. Cores whose file cannot be opened, or whose read
     fails (e.g. `-EBUSY` for an offline CPU's inactive policy), report 0.
- **Record format**: new `StatRecord.CpuFreq *CpuFreqStat` with `NumCore`
  and `CoreFreqs []int64` (kHz, 0 = unavailable). Plain gob field addition:
  old `.pgr` files decode with `CpuFreq == nil`, old binaries ignore the new
  field. No format version bump.
- **play JSON**: new top-level key after `cpu`:
  `"cpufreq": {"num_core": N, "cores": [<MHz>, ...]}` (MHz, `%.1f`;
  `0.0` = unavailable). Omitted when the record has no frequency data.
- **summary / stat**: time-averaged frequency per core (ignoring 0 =
  unavailable samples) and the mean across cores, in both text and JSON
  (`"cpufreq": {"num_core": N, "avg": <MHz>, "cores": [...]}`).
- **CLI**: `--no-cpufreq` for `record`, `live`, `stat` (and forwarded by the
  `record --background` re-exec), plus the legacy recorder flag set and the
  zsh completion.
- **Out of scope** (follow-ups if needed): `plot` support, dedicated `live`
  rendering, recording static cpufreq metadata (driver, governor, min/max)
  in the header (overlaps with #0010), and interval-average frequency via
  APERF/MPERF MSRs (needs root and IPIs).

## Resolution

Implemented as designed above.

- `core/internal/perfmonger/cpufreq_linux.go` (new): `CpuFreqReader`
  (`OpenCpuFreqReader` / `Read` / `Close`). fds are opened `O_CLOEXEC` once
  and re-read with `syscall.Pread(fd, buf, 0)`; parsing is allocation-free.
  The probe opens the files, then applies the kernel gate (x86 && < 5.19 →
  `*CpuFreqKernelError`) **before** any read, then does one probe read
  (no valid value anywhere → `ErrCpuFreqUnavailable`).
- `stat.go`: `CpuFreqStat`, `StatRecord.CpuFreq`; `NewStatRecord` switched to
  a keyed literal. `perfmonger_linux.go`: core-count detection factored out
  into `NumCpuCores()` (behavior unchanged).
- `usage.go`: `CpuFreqStat.WriteJsonTo` (MHz, `%.1f`) and `CpuFreqSummary`
  (per-core time average ignoring 0, mean across cores, min/max).
- recorder: `RecorderOption.NoCpuFreq`, `--no-cpufreq` in the legacy flag set,
  `setupCpuFreqReader` (stderr notice only for the kernel gate; other
  reasons only with `PERFMONGER_DEBUG`), read once per sample.
  `ReportCpuFreqSupport` lets the `record --background` parent print the
  notice, because the detached child's stderr is `/dev/null`.
- player: `"cpufreq"` key after `"cpu"`. summarizer: `"cpufreq"` in JSON and
  an "Average CPU frequency" block in text, averaged over all records.
- CLI: `--no-cpufreq` on `record` / `live` / `stat`, forwarded by the
  background re-exec, zsh completion.
- Docs: README ("Per-core CPU frequency" section, JSON schema, summary
  example) and `doc/architecture.md`.

Tests added:

- Go: `cpufreq_linux_test.go` (fake-sysfs reader incl. missing/garbage cores
  and re-read from offset 0, kernel-version parsing and gate, unavailable
  paths, Close idempotency, real-sysfs smoke test, gob compatibility with a
  pre-change `StatRecord` in both directions, benchmarks), `usage_test.go`
  (JSON writer, summary math), recorder `cpufreq_test.go` (probe messages,
  `NoCpuFreq`, RunDirect with/without frequencies), flag-presence checks in
  `record_test.go` / `stat_test.go`.
- pytest: `tests/test_cpufreq.py` (record→play includes `cpufreq` when usable,
  `--no-cpufreq`, fallback on hosts without cpufreq, legacy fixture has no
  `cpufreq` in play/summary, summary JSON/text average).

Measured on the dev host (Xeon E5-2680, 16 CPUs, kernel 6.8,
`intel_cpufreq` + `schedutil`): `BenchmarkCpuFreqReaderRead` 24.8 µs/sweep,
0 allocs (vs. 263 µs for `ReadCpuStat`). Recording at 100 Hz for 5 s: CPU
time 1061/1071 ms with cpufreq vs. 977/995 ms without (≈ 160 µs/sample with
cold caches), no difference in `CAL`/`RES` IPI counts, +70 bytes/sample.

Independent review (ticket-evaluator): PASS, nits only. Fixed a misplaced
doc comment. Left as is:

- CPU hotplug is untested here (needs root). The fd refers to the cpufreq
  policy's attribute, which on offline turns inactive (`show()` returns
  `-EBUSY`, recorded as 0) and is reused when the CPU comes back, so reads
  should resume. A CPU whose policy did not exist at start-up stays 0 for the
  whole recording.
- `record --background` probes twice (parent for the notice, child for
  recording), and `NumCpuCores()` still spawns `nproc` once more at start-up.
  Both are one-off costs.
- The < 5.19 gate is covered by unit tests with injected arch/release only; it
  cannot be exercised end to end on the 6.8 dev host.

Follow-ups (not filed): `plot` support for frequencies; static cpufreq
metadata (driver/governor/min/max) in the header together with #0010.
Unrelated observation: `make build` never rebuilds an existing
`lib/exec/perfmonger_linux_amd64` (the target has no source prerequisites);
use `make -B build`.
