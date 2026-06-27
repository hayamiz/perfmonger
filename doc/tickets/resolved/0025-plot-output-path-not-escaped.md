---
title: Gnuplot output path embedded in script without escaping, breaking on special characters
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/plot.go`, `generateDiskIOPSPlot` (around lines 362-373,
and the three analogous plot-generating functions) embeds `outFile` directly into
a gnuplot script string via `fmt.Sprintf` with `%s`, e.g. line 364:
`set output "%s"`.

The gnuplot `set output` command uses C-style string escaping. If `cmd.OutputDir`
or the derived `outFile` contains a double-quote, backslash, or newline, the
generated script becomes syntactically invalid and gnuplot emits an error or
writes to the wrong location. Backslashes in Windows-style paths would also be
misinterpreted on cross-platform builds.

Why it is a bug: legitimate output directories containing special characters
silently break plot generation or redirect output to an unintended file.

Suggested direction: escape the path for gnuplot's string syntax (at minimum
backslashes and double-quotes) before interpolating, or use a gnuplot variable /
argument-passing mechanism rather than string concatenation. Apply the same fix to
all four plot-generating functions.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: Output paths are embedded into gnuplot scripts via %s without escaping. Fix: a helper that escapes double-quotes/backslashes per gnuplot C-string rules, applied uniformly at all embed sites.

## Resolution

Resolved via strict TDD.

- Added `escapeGnuplotString` to `core/cmd/perfmonger/plot.go`, which escapes
  backslashes (first) and double-quotes per gnuplot C-style double-quoted string
  rules.
- Applied the helper at every site that embeds the output path into a gnuplot
  script: `generateDiskIOPSPlot`, `generateDiskTransferPlot`, `generateCPUPlot`
  (all `set output "%s"`), and `generateAllCPUPlot` (the `fmt.Fprintf(&sb, "set
  output \"%s\"\n", ...)` line).
- Tests added in `core/cmd/perfmonger/plot_test.go`:
  - `TestEscapeGnuplotString` — table-driven, covers plain paths, double-quote,
    backslash, Windows-style path, and combined quote+backslash.
  - `TestGenerateDiskIOPSPlot_EscapesOutputPath` — asserts a path containing a
    double-quote is escaped.

Verification: `go test -count=1 ./cmd/perfmonger/` passes; `go build` of the
unified binary succeeds. Scope limited to gnuplot output-path escaping only.
