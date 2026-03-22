# TODO: perfmonger 残タスク

完了済みタスクは `devdoc/DONE.md` を参照。

## Known Bugs

- [x] record コマンドの --background オプションが正しく動作しない → **修正済み**
  - fork() を re-exec パターンに置き換え（Go ランタイムは fork 非対応）
  - 出力パスをデーモン化前に絶対パスに変換
  - セッション管理（session file 作成/削除）を recorder.go から record.go CLI 層に移動
  - --kill / --status を Ruby 版互換で実装

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

- [x] live コマンドが起動時にクラッシュする → **修正済み**
  - player binary を `os.Executable() + "play"` subcommand に変更
  - player_stdout が nil の場合の goroutine 起動ガード追加
  - play サブコマンドが stdin からの読み取りに対応

- [x] stat コマンドが panic で異常終了する → **修正済み**
  - `RecorderOption.StopCh` チャネルを追加し、recorder の select ループで監視
  - stat コマンドがユーザーコマンド終了後に `close(stopCh)` で recorder を停止
  - recorder が完全に flush してからファイルを summarizer に渡す

- [x] record の `--no-gzip` オプションだけではデフォルトファイル名が `.pgr.gz` のまま → **修正済み**
  - `applyRubySpecificLogic()` で `--no-gzip` 時にファイル名から `.gz` を除去

- [x] plot コマンドが `perfmonger-core_linux_amd64` バイナリに依存している → **修正済み**
  - hidden `plot-format` サブコマンドを perfmonger バイナリに追加
  - `findPlotFormatterBinary()` で `os.Executable()` を使用
  - `generatePlots()` を完全実装（gnuplot スクリプト生成・実行）

## 段階2: 残タスク

- [x] Ruby 互換セッション管理機能実装（kill/status） → **修正済み**
  - `record.go` の `killSession()` / `showStatus()` / `getRunningSessionPID()` を実装
  - `stat.go` / `live.go` は未対応（別途対応予定）
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
