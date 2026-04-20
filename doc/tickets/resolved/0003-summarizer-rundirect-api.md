---
title: Implement `RunDirect(*SummaryOption)` for summarizer package
type: refactor
priority: medium
status: resolved
created: 2026-04-20
updated: 2026-04-20
---

## Description

`core/cmd/perfmonger-core/summarizer` の `RunWithOption()` も player と同様の二重変換構造になっている。0002 と対の改修。

現状 ([core/cmd/perfmonger-core/summarizer/summarizer.go](core/cmd/perfmonger-core/summarizer/summarizer.go)):
- 66行目付近 `RunWithOption(option *SummaryOption)` が args を組み立て
- 86行目付近で `Run(args)` を呼ぶ
- 94行目付近で `parseArgs(args, option)` により再パースされる

やること:
- `RunDirect(option *SummaryOption) error` を新規追加
- `RunWithOption()` の実装を `RunDirect()` 呼び出しへ置換（互換は維持）
- `core/cmd/perfmonger/summary.go` で `RunDirect()` を直接使うよう変更
- pytest の `test_summary.py` がグリーンであること

関連: [0002-player-rundirect-api.md](0002-player-rundirect-api.md)

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: 0002 と同パターンの機械的適用。pager (#0001) は `summary.go` 層の関心で、summarizer 層のリファクタとは独立しているため先行して進めて問題なし。ただし #0001 で `summarizer.RunDirect` のシグネチャに `io.Writer` を追加する案があるため、両チケットを同時期に進める場合はインターフェイス設計を一度に決めると手戻りが減る。

## Resolution

Added `RunDirect(option *SummaryOption, out io.Writer) error` in
`core/cmd/perfmonger-core/summarizer/summarizer.go`. All previous
stdout writes (`fmt.Println` / `fmt.Printf`, including the JSON and
text rendering branches) were routed through the supplied `io.Writer`
via `fmt.Fprintln` / `fmt.Fprintf`. `RunWithOption` is now a thin
backward-compatible wrapper that calls `RunDirect(option, os.Stdout)`.
`Run([]string)` likewise delegates to `RunDirect` after `parseArgs`.
`RunDirect` defensively compiles `option.DiskOnly` into
`option.DiskOnlyRegex` when the caller has not (cobra sets only the
string form).

`core/cmd/perfmonger/summary.go` was updated so both `run()` and
`runWithPager()` call `summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)`
directly and return the error.

Threading `io.Writer` from the start (per ticket #0001 coordination
note) means #0001 can later redirect summary output to the pager's
stdin without any additional signature change.

Verification:
- `cd core/internal/perfmonger && go test -v -cover` — PASS, coverage 54.2%.
- `cd core/internal/perfmonger && go vet ...` — clean (no diagnostics).
- `uv run pytest -v -k test_summary` — 4 passed, 32 deselected.
- Rebuilt `lib/exec/perfmonger_linux_amd64` with no errors.

Files changed:
- `core/cmd/perfmonger-core/summarizer/summarizer.go`
- `core/cmd/perfmonger/summary.go`
- `doc/tickets/0003-summarizer-rundirect-api.md` (frontmatter + Resolution)
