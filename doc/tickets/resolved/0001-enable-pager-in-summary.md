---
title: Enable pager support in `summary` subcommand
type: feature
priority: medium
status: resolved
created: 2026-04-20
updated: 2026-04-20
---

## Description

`summary` サブコマンドの pager 機能が新 API 移行時に一時的に無効化されたまま残っている。現状の [core/cmd/perfmonger/summary.go](core/cmd/perfmonger/summary.go) の `runWithPager()` は以下のように警告を出すだけのスタブになっている:

```go
fmt.Fprintf(os.Stderr, "Warning: pager functionality temporarily disabled with new API\n")
```

`validatePagerOption` 等で `PAGER` 環境変数の読み取り準備は一部入っているが、実際の pager 呼び出し（パイプ接続・子プロセス起動・wait）が未実装。Ruby 実装時と同等の挙動を取り戻す必要がある。

対象:
- 環境変数 `PAGER` のサポート（未設定時のフォールバックは `less -R` 相当を想定）
- 出力が TTY でないとき（リダイレクトやパイプ）は pager を介さず直接 stdout に書く
- pager プロセスの stdin にサマリ出力をストリーミングし、終了を wait
- `--no-pager` 相当のオプションがあれば尊重（既存フラグを確認）

完了条件:
- `runWithPager()` のスタブ警告が消える
- TTY 実行時に `PAGER=cat perfmonger summary ...` がエラー無く動作
- パイプ実行時（`perfmonger summary ... | head`）に pager を起動しないことを pytest 統合テストで確認

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Implementation Notes の Resolution でユーザ決定が確定済み（PAGER 未設定/空文字 → stdout 直書き、pager 実行不能時は警告+stdout フォールバック、`--no-pager` フラグ追加、ANSIカラー予定なし）。残作業は `summary.go` に TTY 検出・`resolvePager`・`pipeToPager`（`os/exec` Cmd.StdinPipe + Wait）・`--no-pager` フラグを実装し、summarizer 側は `io.Writer` を受け取る形へ拡張（#0003 と整合）。

## Implementation Notes

### 触るファイル

- [core/cmd/perfmonger/summary.go](core/cmd/perfmonger/summary.go) — `runWithPager()` の実装、必要なら `--no-pager` フラグ追加
- [core/cmd/perfmonger-core/summarizer/summarizer.go](core/cmd/perfmonger-core/summarizer/summarizer.go) — 出力先を `io.Writer` 引数で受け取れるように拡張（現状 `fmt.Print*` で stdout 直書き）

### 提案シグネチャ

```go
// summarizer 側で書き込み先を引数化
func RunDirect(option *SummaryOption, out io.Writer) error

// summary.go 側
func (cmd *summaryCommand) runWithPager() error {
    pager := cmd.resolvePager()      // --pager / PAGER / default の順
    if pager == "" || !isTTY(os.Stdout) {
        return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
    }
    return cmd.pipeToPager(pager)    // Cmd.StdinPipe → Wait, シグナル伝播
}
```

### 要判断事項（ユーザ）

1. **デフォルト pager**: `PAGER` 未設定・`--pager` 未指定時の既定は `less -R` / `cat` / 「pager なし（stdout 直書き）」のどれか？
2. **`PAGER=""` の扱い**: pager 無効化として stdout 直書きか、Ruby 実装準拠でエラーにするか？
3. **pager 実行不能時**: 明示エラーで終了 or 警告を出して stdout フォールバック？
4. **`--no-pager` フラグ**: `git --no-pager` 風に追加するか、既存の `--pager` だけで足りるとするか？
5. **色出力**: summary 出力に ANSI カラーを追加する予定があるか（あれば `less -R` が既定として妥当）？

### pytest テスト観点

- `PAGER=cat` で TTY 相当の状態（pty 経由 or モック）にしたとき出力が通る
- パイプ接続時（`perfmonger summary ... | head`）で pager が起動しない
- `PAGER=/nonexistent` の時に選択した失敗モードで挙動する
- `--no-pager` 追加時、`PAGER` 設定下でも stdout 直書きになる

## Resolution

### Decided behavior

- PAGER未設定時は pagerなし（stdout直書き）を既定とする
- PAGER="" も pagerなし（stdout直書き）とする
- pager実行不能時は警告を出して stdout 直書きにフォールバック
- `--no-pager` フラグを追加して pager を明示的に無効化できるようにする
- ANSIカラーは現状予定なし

### Changes

- [core/cmd/perfmonger/summary.go](../../core/cmd/perfmonger/summary.go):
  reimplemented `runWithPager()` with real pager logic; added `--no-pager`
  bool flag; removed the `validatePagerOption` stub and the
  "pager temporarily disabled" warning. The new flow:
  1. `--no-pager` short-circuits to direct stdout.
  2. Resolve pager from `--pager` flag (priority) or `PAGER` env.
  3. If no pager or stdout is not a TTY → stdout.
  4. Otherwise split the pager string with `strings.Fields` (supports
     `PAGER="less -R"`), `exec.Command`, attach `StdinPipe`, `Start`. On
     start failure, emit a `Warning:` to stderr and fall back to stdout.
  5. Stream the summarizer output into the pager's stdin, close the pipe,
     `Wait`. Broken-pipe write errors (user pressed `q`) are swallowed.
- [core/cmd/perfmonger-core/summarizer/summarizer.go](../../core/cmd/perfmonger-core/summarizer/summarizer.go):
  `RunDirect(option *SummaryOption, out io.Writer) error` now writes to the
  supplied `io.Writer`; `RunWithOption` and `Run` are thin wrappers over
  `RunDirect` with `os.Stdout` (per #0003's plan, which had not actually
  been merged when this ticket was picked up).
- [core/go.mod](../../core/go.mod) already listed `golang.org/x/term` as a
  direct require; no module changes needed. The code uses
  `term.IsTerminal(int(os.Stdout.Fd()))` for TTY detection.

### Tests added

Integration tests in
[tests/test_summary.py](../../tests/test_summary.py):

- `test_summary_piped_stdout_does_not_use_pager` — `PAGER=cat` + piped
  stdout: summary lands on stdout, no pager warning on stderr.
- `test_summary_no_pager_flag_disables_pager` — `--no-pager` with
  `PAGER=cat` still writes directly to stdout.
- `test_summary_pager_unset_writes_to_stdout` — `PAGER` unset in env:
  direct stdout.
- `test_summary_pager_empty_writes_to_stdout` — `PAGER=""`: direct stdout,
  no warning.
- `test_summary_pager_nonexistent_with_pipe_still_works` —
  `PAGER=/does/not/exist/pager` with piped stdout: pager path isn't
  attempted (non-TTY), so no warning and summary is on stdout.

The "pager exec fails on a real TTY → warning + fallback" branch was
verified manually using `pty.fork()` (Python) to spawn the binary with a
PTY attached to stdout: setting `PAGER=/nonexistent/pager` produced the
expected `Warning: failed to start pager ...` on stderr and the summary
on stdout with exit code 0. Automating this in pytest would require a
PTY fixture, which is out of scope for this ticket.

### Verification

- `cd core/internal/perfmonger && go test -v -cover` → PASS (15 tests,
  54.2% coverage).
- `cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go |
  grep -v perfmonger_)` → clean.
- `uv run pytest -v -k test_summary` → 9/9 PASS (4 pre-existing + 5 new).
