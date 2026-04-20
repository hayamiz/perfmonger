---
title: Implement `RunDirect(*PlayerOption)` for player package
type: refactor
priority: medium
status: resolved
created: 2026-04-20
updated: 2026-04-20
---

## Description

`core/cmd/perfmonger-core/player` の `RunWithOption()` が内部で option → args 配列 → `Run(args)` → `parseArgs(args, option)` の二重変換を行っている。recorder / plotformatter では既に `RunDirect(option)` 方式への移行が完了しており ([doc/DONE.md](doc/DONE.md) 段階2.5–2.8)、player だけ取り残されている。

現状 ([core/cmd/perfmonger-core/player/player.go](core/cmd/perfmonger-core/player/player.go)):
- 192行目付近の `RunWithOption(option *PlayerOption)` が args を組み立て
- 212行目付近で `Run(args)` を呼ぶ
- `Run()` が再度 `parseArgs(args, option)` でパースし直している

やること:
- `RunDirect(option *PlayerOption) error` を新規追加し、args 生成・再パースを経由せず `option` をそのまま使って再生処理に入る
- `RunWithOption()` は互換のため残すが内部実装を `RunDirect()` 呼び出しに置換
- `core/cmd/perfmonger/play.go` 側で `RunDirect()` を直接呼ぶよう修正
- 既存の Go 単体テスト（`playCommand` の validate 系）と pytest の `test_play.py` がグリーンであること

関連 DONE:
- recorder の RunDirect API 実装（段階2.5–2.8）
- plotformatter の RunDirect API 実装（PlotFormatOption 公開構造体）

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: recorder / plotformatter で既に確立したパターンを player に当てはめるだけ。`RunDirect(option)` を新設し `RunWithOption` はその薄いラッパーに置換、`play.go` の呼び出しを `RunDirect` に差し替える。ロジック変更なし、既存テストも変更不要の見込み。

## Resolution

Applied the recorder.go pattern to the player package:

- `core/cmd/perfmonger-core/player/player.go`:
  - Added `RunDirect(option *PlayerOption)` containing the actual playback logic (file open, gob decode, record loop).
  - `RunWithOption(option)` is now a thin wrapper that simply calls `RunDirect(option)` — no more option → args → reparse round-trip.
  - `Run(args)` still supports legacy callers: it creates a default option, runs `parseArgs(args, option)`, then delegates to `RunDirect(option)`.
- `core/cmd/perfmonger/play.go`: the cobra `playCommand.run()` now calls `player.RunDirect(cmd.PlayerOpt)` directly.

Pure refactor, no behavior change, no new tests required.

Verification:
- `cd core/internal/perfmonger && go test -v -cover` — all 15 tests PASS, coverage 54.2%.
- `cd core/internal/perfmonger && go vet ...` — clean.
- `uv run pytest tests/test_play.py -v` — 3/3 PASS.
- Unified binary builds cleanly at `lib/exec/perfmonger_linux_amd64`.

Note: `tests/test_background.py` has pre-existing failures on master (caused by `user.Current()` returning "unknown" in this container environment); confirmed present before this refactor and unrelated to player.
