# TODO: perfmonger 残タスク

完了済みタスクは `devdoc/DONE.md` を参照。

## Known Bugs

現在未解決のバグはなし。

## 残タスク

### Go 単体テストの修正

- [x] `*_test.go` を新しい構造体に合わせて書き直し済み
  - `recordCommand`, `playCommand`, `statCommand`, `summaryCommand` の validateOptions / validateAndSetLogfile / validateAndSetCommand をテスト
  - `go test -v -count=1` で全30テストケース PASS

### live / stat の --kill / --status は削除済み

- [x] `live.go` / `stat.go` では --kill / --status はサポート不要のため削除
  - バックグラウンド記録のセッション管理は record コマンドのみの機能

### pager 機能の有効化

- [ ] `summary.go` の `runWithPager()` が一時的に無効化されている
  - 「Warning: pager functionality temporarily disabled with new API」を出力するだけ
  - 環境変数 `PAGER` のサポートを実装する

### player / summarizer の RunDirect API 実装

- [ ] player: `RunWithOption()` が内部で args 変換 → `Run(args)` → `parseArgs` の二重変換を行っている
  - `RunDirect(option *PlayerOption)` を実装して二重変換を排除する
- [ ] summarizer: 同上
  - `RunDirect(option *SummaryOption)` を実装する

### Ruby コードの撤去

- [ ] 以下のファイル・ディレクトリがすべて残存しており、撤去が必要
  - `lib/perfmonger/` — Ruby CLI 層（Go に移行済み）
  - `exe/perfmonger` — Ruby エントリポイント
  - `Gemfile`, `Gemfile.lock` — Ruby 依存管理
  - `perfmonger.gemspec` — gem パッケージ定義
  - `spec/*.rb` — RSpec テスト（pytest に移行済み）
  - `Rakefile` — Ruby ベースのビルド/テストタスク
- [ ] README.md を Go 単一バイナリ前提の内容に更新

### ビルド/配布の Go 単体化

- [ ] `Rakefile` の Ruby 依存ビルド・テストタスクを `Makefile` または `go` コマンド直接に置換
- [ ] クロスビルド・リリース生成の仕組みを整備
