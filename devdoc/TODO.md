# TODO: perfmonger 残タスク

完了済みタスクは `devdoc/DONE.md` を参照。

## Known Bugs

現在未解決のバグはなし。

## 残タスク

### Go 単体テストの修正

- [ ] `*_test.go` がコンパイルエラーになる
  - `record_test.go`, `play_test.go`, `stat_test.go`, `summary_test.go` が段階2.6-2.8のリファクタリングで削除された構造体・関数を参照している
  - 参照先: `newPlayOptions`, `newRecordOptions`, `newSummaryOptions`, `newStatOptions`, `buildRecorderArgs`, `buildSummaryArgs`, `playOptions` 等
  - `go test ./...` がビルドエラーで失敗する
  - 対応: 新しい構造体（`recordCommand`, `playCommand` 等）に合わせてテストを書き直すか、不要なテストを削除

### live / stat の kill / status 実装

- [ ] `live.go` の `killSession()` / `showStatus()` がスタブのまま
  - 「kill functionality not yet implemented」を返すだけ
  - `record.go` の実装を共通化して適用する
- [ ] `stat.go` の `killSession()` / `showStatus()` がスタブのまま
  - 同上

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
