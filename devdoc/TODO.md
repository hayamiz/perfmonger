# TODO: perfmonger 残タスク

完了済みタスクは `devdoc/DONE.md` を参照。

## Known Bugs

- [ ] record コマンドの --background オプションが正しく動作しない
  - `daemonize()` で `os.Chdir("/")` するが、ログファイルパスを事前に絶対パスに変換していない
  - 相対パス（例: `perfmonger.pgr`）が `/perfmonger.pgr` に書き込まれてしまう

  ### Ruby版の --background / --kill / --status 仕様（修正時の参考）

  #### 関連オプション
  | オプション | 説明 |
  |---|---|
  | `--background` | バックグラウンドで記録開始 |
  | `--kill` | 実行中のバックグラウンド記録を停止 |
  | `--status` | 実行中のバックグラウンド記録の状態表示 |

  #### セッション管理
  - セッションファイル: `/tmp/perfmonger-<username>-session.pid` に実行中 PID を記録
  - ロックファイル: `/tmp/.perfmonger.lock` で排他制御
  - Go 側 `recorder.go` の `handleBackgroundSession()` にセッションファイル管理の実装あり

  #### `--background` の動作フロー（Ruby版 `record.rb`）
  1. セッションファイルから既存 PID を読み取り
  2. PID のプロセスが生存しているか確認 → 生存していればエラー終了（多重起動防止）
  3. `Process.daemon(true)` でデーモン化（**cwd を変更しない**）
  4. Go の recorder バイナリを `Process.exec` で起動（自プロセスを置き換え）

  #### `--kill` の動作フロー（Ruby版 `record.rb`）
  1. セッションファイルの PID に `SIGINT` 送信
  2. 指数バックオフ（0.05s→0.1s→...、最大5回）でプロセス終了を待機
  3. 終了しなければエラー。セッションファイルの PID が無効な場合はファイル削除

  #### `--status` の動作フロー（Ruby版 `record.rb`）
  1. `/proc/<PID>/cmdline` から実行コマンド取得
  2. `/proc/<PID>` の mtime から起動時刻取得
  3. PID・実行ファイル・引数・起動時刻・経過時間を表示

  #### Go 実装の問題点
  1. **cwd 変更**: Go 版 `daemonize()` が `os.Chdir("/")` する。Ruby 版は `Process.daemon(true)` で cwd を保持
  2. **セッション管理の分断**: `recorder.go` に `handleBackgroundSession()` があるが、CLI 層 `record.go` の `killSession()` / `showStatus()` はスタブ
  3. **デーモン化タイミング**: Ruby 版は recorder exec 前にデーモン化。Go 版は自プロセス内でデーモン化後に recording 開始

  #### 関連ファイル
  - Ruby 版: `lib/perfmonger/command/record.rb`, `lib/perfmonger/command/record_option.rb`
  - Go CLI 層: `core/cmd/perfmonger/record.go`
  - Go コア層: `core/cmd/perfmonger-core/recorder/recorder.go` (`handleBackgroundSession()`, `RunDirect()`)

- [ ] live コマンドが起動時にクラッシュする
  - recorder の `RunDirect()` 内で nil pointer dereference (SIGSEGV) が発生
  - `recorder.go:333` 付近の goroutine で発生
  - テスト: `tests/test_live.py::test_live_outputs_json_and_creates_logfile` (xfail)

- [ ] stat コマンドが panic で異常終了する
  - summarizer の `Run()` 内で `unexpected EOF` panic が発生
  - `summarizer.go:110` 付近
  - recorder がまだ実行中の可能性を示す warning も出力される
  - テスト: `tests/test_stat.py::test_stat_creates_logfile` (xfail)

- [ ] record の `--no-gzip` オプションだけではデフォルトファイル名が `.pgr.gz` のまま
  - Ruby 版ではデフォルトが `perfmonger.pgr` で、`--gzip` 時に自動で `.gz` を付与していた
  - Go 版ではデフォルトが `perfmonger.pgr.gz` で、`--no-gzip` でもファイル名が変わらない
  - `-l perfmonger.pgr` を明示すれば動作する
  - 関連ファイル: `core/cmd/perfmonger/record.go`

- [ ] plot コマンドが `perfmonger-core_linux_amd64` バイナリに依存している
  - plot は内部で `perfmonger-core_linux_amd64 plot-format` を fork/exec する
  - `lib/exec/` にシンボリックリンクはあるがリンク先の `perfmonger-core_linux_amd64` が存在しない
  - `core/build.sh` で perfmonger-core もビルドするか、plot の実装を perfmonger 単一バイナリ内に統合する必要がある
  - テスト: `tests/test_plot.py` の全4テスト (xfail)

## 段階2: 残タスク

- [ ] Ruby 互換セッション管理機能実装（kill/status）
  - `record.go` の `killSession()` / `showStatus()` が "not yet implemented" のスタブ状態
  - `stat.go` / `live.go` も同様
  - `getRunningSessionPID()` も未実装（常に 0 を返す）
- [ ] Go 側統合テスト追加（`spec/data/` のサンプルを golden として活用）
- [ ] ビルド/配布を Go 単体に一本化（クロスビルド・リリース生成）
  - Go バイナリは動作するが、Rakefile のビルド/テスト実行が Ruby 依存
- [ ] Ruby コード・gemspec の撤去、README 更新
  - lib/perfmonger/, exe/, Gemfile, perfmonger.gemspec, spec/*.rb 等が全て残存

## 段階2.5: 残タスク

- [ ] plot/plotformatter の詳細最適化（簡略化実装のまま）
- [ ] 完全な pager 統合サポート（summary コマンド）
  - `runWithPager()` が "Warning: pager functionality temporarily disabled" のスタブ状態
- [ ] 単体テストの更新
  - `stat_test.go` / `record_test.go` に削除済みの `buildXXXArgs()` 関数のテストが残存
  - これらのテストはコンパイルエラーになる可能性あり

## 段階2.9: コアコンポーネントの直接実行API統一

### 目標: player/summarizerコンポーネントにもRunDirect APIを実装

#### 現在の状況
- recorder: ✅ RunDirect実装済み（段階2.7で完了）
- player: ❌ Run(args)のみ、RunWithOptionがargs変換を行う
- summarizer: ❌ Run(args)のみ、RunWithOptionがargs変換を行う
- plotformatter: ❌ Run(args)のみ、構造体APIなし

#### 実装計画

##### 1. player コンポーネント [優先度: 高]
- [ ] `RunDirect(option *PlayerOption)`関数の実装
- [ ] RunWithOptionをRunDirect呼び出しに変更
- [ ] Run(args)は互換性のため維持

##### 2. summarizer コンポーネント [優先度: 高]
- [ ] `RunDirect(option *SummaryOption)`関数の実装
- [ ] RunWithOptionをRunDirect呼び出しに変更
- [ ] Run(args)は互換性のため維持

##### 3. plotformatter コンポーネント [優先度: 中]
- [ ] `PlotFormatterOption`構造体の公開化
- [ ] `NewPlotFormatterOption()`関数の追加
- [ ] `RunWithOption(option *PlotFormatterOption)`関数の新規実装
- [ ] `RunDirect(option *PlotFormatterOption)`関数の実装
- [ ] plot.goからの直接呼び出し対応

## 段階3: テスト基盤の移行（RSpec → Bats）

- [ ] **RSpec テスト基盤の置き換え**
  - [ ] Bats (Bash Automated Testing System) の導入
    - [ ] Bats のインストール・セットアップ手順をドキュメント化
    - [ ] CI/CD パイプラインに Bats テスト実行を統合
  - [ ] `./spec` 以下の RSpec テストを Bats に移行
    - [ ] `spec/record_spec.rb` → Bats テスト化
    - [ ] `spec/play_spec.rb` → Bats テスト化
    - [ ] `spec/summary_spec.rb` → Bats テスト化
    - [ ] `spec/stat_spec.rb` → Bats テスト化
    - [ ] `spec/plot_spec.rb` → Bats テスト化
    - [ ] `spec/fingerprint_spec.rb` → Bats テスト化
    - [ ] `spec/live_spec.rb` → Bats テスト化
  - [ ] テストデータとヘルパー関数の移行
    - [ ] `spec/data/` 以下のテストデータファイルを活用
    - [ ] `spec_helper.rb` の機能を Bats ヘルパー関数として実装
  - [ ] テスト実行の統合
    - [ ] `Rakefile` から Bats テスト実行タスクを呼び出し
    - [ ] Ruby 依存完全削除後は Bats のみでテスト実行
