---
title: Implement `RunDirect(*PlayerOption)` for player package
type: refactor
priority: medium
status: open
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
