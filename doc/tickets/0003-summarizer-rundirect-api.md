---
title: Implement `RunDirect(*SummaryOption)` for summarizer package
type: refactor
priority: medium
status: open
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
