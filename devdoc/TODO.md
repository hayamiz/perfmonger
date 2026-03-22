# TODO: perfmonger 残タスク

完了済みタスクは `devdoc/DONE.md` を参照。

## Known Bugs

- [ ] record コマンドの --background オプションが正しく動作しない
  - `daemonize()` で `os.Chdir("/")` するが、ログファイルパスを事前に絶対パスに変換していない
  - 相対パス（例: `perfmonger.pgr`）が `/perfmonger.pgr` に書き込まれてしまう

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
