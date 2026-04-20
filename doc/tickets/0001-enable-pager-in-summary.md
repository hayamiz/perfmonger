---
title: Enable pager support in `summary` subcommand
type: feature
priority: medium
status: open
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
