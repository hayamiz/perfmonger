# TODO: perfmonger 構成移行プラン（段階1 → 段階2）

目的: 機能互換を維持しつつ、(1) core の複数バイナリを単一 `perfmonger-core` に集約し、(2) Ruby CLI を廃止して Go 実装へ完全移行し単一 `perfmonger` バイナリに統合する。

## 段階0: 不要な機能の整理、コード構造の整理

### タスク1: Darwin サポートの廃止

- [x] `core/subsystem` 以下の darwin サポートを廃止して、darwin関連のコードを削除
  - [x] `core/subsystem/perfmonger_darwin.go` を削除
  - [x] `core/subsystem/perfmonger.go` から Darwin 関連の条件分岐を削除
  - [x] `core/build.sh` から Darwin ビルドターゲットを削除（30行目の "darwin amd64" を削除）
  - [x] `core/Makefile` から Darwin ビルドターゲットを削除（自動生成されるため build.sh の修正で対応）
  - [x] `Rakefile` の analyze_core タスク（45行目）から "darwin" を削除
  - [x] `lib/perfmonger/command/core.rb` の CoreFinder から Darwin 関連のロジックを削除
  - [x] `README.md` から Mac OS X サポートの記述を削除（19行目）
  - [x] テストを実行して Linux ビルドが正常に動作することを確認

### タスク2: パッケージ構造のリファクタリング

- [ ] `core/subsystem` 以下のパッケージを、`core/cmd` 以下から参照するときに `github.com/hayamiz/perfmonger/core/subsystem` として参照する方式をやめて、`core/subsystem` 以下のパッケージを `core/cmd` 以下に移動して、直接参照する方式に変更
  - [ ] 現在の import 構造の確認（4つの cmd が subsystem を参照）:
    - `perfmonger-player`: subsystem.StatRecord を使用
    - `perfmonger-plot-formatter`: subsystem.StatRecord を使用  
    - `perfmonger-recorder`: subsystem.NewPerfMonger を使用
    - `perfmonger-summarizer`: subsystem.StatRecord を使用
  - [ ] `core/internal/perfmonger` ディレクトリを作成
  - [ ] `core/subsystem/*.go` を `core/internal/perfmonger/` に移動（テストファイル含む）
  - [ ] 各 cmd の import を更新:
    - `"github.com/hayamiz/perfmonger/core/subsystem"` → `"../internal/perfmonger"`
  - [ ] `core/utils.go` を `core/internal/perfmonger/utils.go` に移動
  - [ ] `go.mod` の module path は変更せず維持（外部からの参照は不要のため）
  - [ ] `Rakefile` の test_core, analyze_core タスクのパスを更新
  - [ ] ビルドとテストを実行して正常動作を確認

## 段階1: core を単一バイナリ化（`perfmonger-core`）

- [ ] 設計確定: 単一バイナリでのサブコマンド/argv[0] 互換方針（ラッパースクリプトで互換、argv[0] でも解釈）
- [ ] 新規エントリ `core/cmd/perfmonger-core` を作成し、サブコマンド `record|play|summarizer|plot-formatter|viewer` を実装（引数互換）
- [ ] 既存各 main のロジックをパッケージ化して再利用
  - [ ] recorder: `Run(args []string)` を公開（元: `core/cmd/perfmonger-recorder/perfmonger-recorder.go:1`）
  - [ ] player  : `Run(args []string)` を公開（元: `core/cmd/perfmonger-player/perfmonger-player.go:1`）
  - [ ] summarizer / plot-formatter / viewer も同様に切り出し
- [ ] 旧各 main は一時的にビルド対象外化（既存ソースは保持）
- [ ] `core/build.sh:1` を修正し `perfmonger-core_<os>_<arch>` のみビルド
- [ ] `lib/exec/` に互換名を配置
  - [ ] `perfmonger-recorder_*`, `perfmonger-player_*`, `perfmonger-summarizer_*`, `perfmonger-plot-formatter_*` のラッパー → `perfmonger-core_*`
- [ ] Ruby 側の互換性検証（`lib/perfmonger/command/core.rb:1` 経由の解決が継続動作）
- [ ] 既存 RSpec/Go テストが通ること（`bundle exec rake`）
- [ ] ドキュメント更新（`AGENTS.md:1`/`README.md:1` に新構成を追記）

### 補足タスク（段階1・内部参照の明確化）

- [ ] `core/app/**` を `core/internal/app/**` に移動し、`perfmonger-core` からは `github.com/hayamiz/perfmonger/core/internal/app/...` を参照するように変更

## 段階2: Ruby を廃止し Go へ完全移行（単一 `perfmonger`）

- [ ] CLI 設計: Go によるトップレベル `perfmonger` コマンドとサブコマンド群を定義
  - live / record / play / summary / plot / stat / server / fingerprint / init-shell
- [ ] Ruby 実装との差分洗い出し（特に `server`, `plot`, `fingerprint`, `init-shell`）
- [ ] 実装（既存 core を活用）
  - [ ] record: 既存 recorder を統合
  - [ ] play: 既存 player を統合
  - [ ] summary: 既存 summarizer を統合
  - [ ] plot: Ruby の gnuplot スクリプト生成を Go へ移植（pdfcairo 検出、ImageMagick 連携オプション）
  - [ ] live: recorder と player を内部パイプで接続
  - [ ] stat: 子プロセス実行＋記録＋サマリの統合処理
  - [ ] server: Ruby/WEBrick + SSE を Go `net/http` + SSE で移植（`data/assets` は `embed` で内包。互換パス: `/dashboard`, `/assets`, `/faucet`）
  - [ ] fingerprint: Ruby 実装を調査・移植
  - [ ] init-shell: Ruby 実装を調査・移植
- [ ] Go 側統合テスト追加（`spec/data/` のサンプルを golden として活用）
- [ ] 一時的に RSpec で互換性を検証 → Go テストへ段階的移行
- [ ] ビルド/配布を Go 単体に一本化（クロスビルド・リリース生成）
- [ ] Ruby コード・gemspec の撤去、README/AGENTS 更新
- [ ] 互換ラッパー（任意）: 旧コマンド名で起動した場合に `perfmonger <subcmd>` を呼ぶスクリプトを配布
- [ ] 完了項目を `devdoc/DONE.md:1` へ移し、移行メモ/既知の差分を記録

## マイルストーン/完了条件

- 段階1完了: Ruby CLI + `CoreFinder` で全サブコマンドが旧名で動作し、RSpec/Go テストがパス。`lib/exec` は単一バイナリ由来。
- 段階2完了: Ruby 依存が無く、`perfmonger` 単一バイナリで同等機能（server/plot 含む）。Go テストがパスし、利用手順が README に反映。
