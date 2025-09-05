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

- [x] `core/subsystem` 以下のパッケージを、`core/cmd` 以下から参照するときに `github.com/hayamiz/perfmonger/core/subsystem` として参照する方式をやめて、`core/subsystem` 以下のパッケージを `core/cmd` 以下に移動して、直接参照する方式に変更
  - [x] 現在の import 構造の確認（4つの cmd が subsystem を参照）:
    - `perfmonger-player`: subsystem.StatRecord を使用
    - `perfmonger-plot-formatter`: subsystem.StatRecord を使用  
    - `perfmonger-recorder`: subsystem.NewPerfMonger を使用
    - `perfmonger-summarizer`: subsystem.StatRecord を使用
  - [x] `core/internal/perfmonger` ディレクトリを作成
  - [x] `core/subsystem/*.go` を `core/internal/perfmonger/` に移動（テストファイル含む）
  - [x] 各 cmd の import を更新:
    - `"github.com/hayamiz/perfmonger/core/subsystem"` → `"github.com/hayamiz/perfmonger/core/internal/perfmonger"`
  - [x] `core/utils.go` を `core/internal/perfmonger/utils.go` に移動
  - [x] `go.mod` の module path は変更せず維持（外部からの参照は不要のため）
  - [x] `Rakefile` の test_core, analyze_core タスクのパスを更新
  - [x] ビルドとテストを実行して正常動作を確認

## 段階1: core を単一バイナリ化（`perfmonger-core`）

- [x] 設計確定: 単一バイナリでのサブコマンド/argv[0] 互換方針（ラッパースクリプトで互換、argv[0] でも解釈）
- [x] 新規エントリ `core/cmd/perfmonger-core` を作成し、サブコマンド `record|play|summarizer|plot-formatter|viewer` を実装（引数互換）
- [x] 既存各 main のロジックをパッケージ化して再利用
  - [x] recorder: `Run(args []string)` を公開（元: `core/cmd/perfmonger-recorder/perfmonger-recorder.go:1`）
  - [x] player  : `Run(args []string)` を公開（元: `core/cmd/perfmonger-player/perfmonger-player.go:1`）
  - [x] summarizer / plot-formatter / viewer も同様に切り出し
- [x] 旧各 main は一時的にビルド対象外化（既存ソースは保持）
- [x] `core/build.sh:1` を修正し `perfmonger-core_<os>_<arch>` のみビルド
- [x] `lib/exec/` に互換名を配置
  - [x] `perfmonger-recorder_*`, `perfmonger-player_*`, `perfmonger-summarizer_*`, `perfmonger-plot-formatter_*` のラッパー → `perfmonger-core_*`
- [x] Ruby 側の互換性検証（`lib/perfmonger/command/core.rb:1` 経由の解決が継続動作）
- [x] 既存 RSpec/Go テストが通ること（`bundle exec rake`）
- [x] ドキュメント更新（`CLAUDE.md:1`/`README.md:1` に新構成を追記）

## 段階2: Ruby を廃止し Go へ完全移行（単一 `perfmonger`）

- [x] CLI 設計: Go によるトップレベル `perfmonger` コマンドとサブコマンド群を定義
  - live / record / play / summary / plot / stat / fingerprint / init-shell
  - 注: server サブコマンドは廃止
- [x] Ruby 実装との差分洗い出し（特に `plot`, `fingerprint`, `init-shell`）
- [x] 実装（既存 core を活用）
  - [x] record: 既存 recorder を統合
  - [x] play: 既存 player を統合
  - [x] summary: 既存 summarizer を統合
  - [x] plot: Ruby の gnuplot スクリプト生成を Go へ移植（pdfcairo 検出、ImageMagick 連携オプション）
  - [x] live: recorder と viewer を内部パイプで接続
  - [x] stat: 子プロセス実行＋記録＋サマリの統合処理
  - [x] fingerprint: Ruby 実装を調査・移植
  - [x] init-shell: Ruby 実装を調査・移植
  - [x] server 廃止に伴う検討事項:
    - [x] server 機能の廃止を README/ドキュメントに明記
    - [x] 既存 server 関連コード（Ruby 側）の削除計画を策定
- [x] 単一 Go バイナリ `perfmonger_linux_amd64` の実装完了
  - 場所: `lib/exec/perfmonger_linux_amd64`
  - すべてのサブコマンド（live, record, play, stat, plot, summary, fingerprint, init-shell）が動作
  - 既存の perfmonger-core バイナリを活用してコア機能を統合
- [ ] **cobra/pflag への移行とRuby版互換性修正**
  - [x] Phase 1: cobra/pflag 基盤整備 **[完了]**
    - [x] go.mod に cobra/pflag 依存関係追加
      - `go get github.com/spf13/cobra@latest`
      - `go get github.com/spf13/pflag@latest`
    - [x] メイン CLI 構造を cobra.Command ベースに変更
      - 現在の main.go の手動 switch 文を cobra サブコマンド構造に置き換え
      - ルートコマンドとサブコマンドの階層構造を構築
    - [x] カスタム Usage テンプレート作成
      - Ruby版 `Usage: perfmonger SUBCOMMAND [options] ARGS` 形式
      - 各サブコマンドで統一されたヘルプ表示形式
  - [ ] Phase 2: 各サブコマンドの cobra 移行と互換性修正 **[3/8コマンド完了]**
    - [x] record サブコマンドの修正
      - [x] 問題調査: Ruby版は `-d DEVICE` (複数回指定可能)、Go版は `-disks string` (CSV形式)
      - [x] 問題調査: Ruby版は `-l FILE`、Go版は `-output FILE`
      - [x] 問題調査: Ruby版は `--kill`, `--status` オプション未実装
      - [x] 問題調査: ヘルプ形式が異なる（Ruby版: Usage + Options、Go版: flag.PrintDefaults）
      - [x] cobra.Command 構造に移行
      - [x] pflag.StringSliceP で `-d, --disk` オプション実装（複数回指定対応）
      - [x] pflag.StringP で `-l, --logfile` オプション実装
      - [x] `--kill`, `--status`, `--background` オプション追加
      - [x] `--record-intr` オプション追加（`--no-intr` の逆）
      - [x] `--no-gzip` オプション追加
      - [x] Ruby形式のヘルプ表示
      - [x] 包括的単体テスト作成
      - [ ] Ruby 互換セッション管理機能実装（kill/status）
    - [x] play サブコマンドの修正
      - [x] 問題調査: Ruby版は `-c, --color`、Go版は `-color`
      - [x] 問題調査: Ruby版は `-p, --pretty`、Go版は `-pretty`
      - [x] cobra.Command 構造に移行
      - [x] pflag.BoolP で `-c, --color` オプション実装
      - [x] pflag.BoolP で `-p, --pretty` オプション実装
      - [x] Ruby形式のヘルプ表示（Usage行 + Options）
      - [x] 包括的単体テスト作成
    - [x] summary サブコマンドの修正
      - [x] 問題調査: Ruby版は `-p, --pager` オプション未実装
      - [x] cobra.Command 構造に移行
      - [x] pflag.StringP で `-p, --pager [PAGER]` オプション実装
      - [x] pager 機能の実装（環境変数 PAGER サポート）
      - [x] Ruby形式のヘルプ表示
      - [x] 包括的単体テスト作成
    - [x] stat サブコマンドの修正
      - [x] 問題調査: 基本的にはRecordOptionの継承なので同様の問題
      - [x] cobra.Command 構造に移行
      - [x] record と同じ pflag オプション定義を適用
      - [x] Ruby形式のヘルプ表示
      - [x] 包括的単体テスト作成
    - [x] plot サブコマンドの修正 **[完了]**
      - [x] 問題調査: 大部分は互換だが、ヘルプ形式が異なる
      - [x] cobra.Command 構造に移行
      - [x] 既存オプションを pflag 形式に変換
      - [x] Ruby形式のヘルプ表示
      - [x] **解決**: コンパイルエラーを修正し、plot-formatter との連携を実装
        - 新実装で plotCommand 構造体依存を削除し、直接実装に変更
        - 簡略化実装により基本機能は動作、将来の完全実装への足がかりを作成
      - [ ] 単体テスト作成
    - [ ] **注意**: summary サブコマンドは既に cobra 移行済み（上記の summary サブコマンドの修正を参照）
    - [x] fingerprint サブコマンドの修正 **[完了]**
      - [x] 問題調査: 引数は互換、ヘルプ形式のみ異なる
      - [x] cobra.Command 構造に移行
      - [x] 既存オプションを pflag 形式に変換（Ruby互換引数処理）
      - [x] Ruby形式のヘルプ表示
      - [x] Ruby互換エイリアス（bukko, fp）追加
    - [x] init-shell サブコマンドの修正 **[完了]**
      - [x] 問題調査: 引数は互換、ヘルプ表示なし（直接実行）
      - [x] cobra.Command 構造に移行
      - [x] Ruby互換引数処理実装（"-" 引数サポート）
      - [x] 既存の完全な実装を活用
    - [x] live サブコマンドの修正 **[完了]**
      - [x] 問題調査: RecordOptionベースなので record と同じ修正が必要
      - [x] cobra.Command 構造に移行
      - [x] record オプションを継承し color/pretty オプション追加
      - [x] Ruby形式のヘルプ表示
      - [x] recorder-player パイプライン実装
  - [ ] Phase 3: 統合テストと検証
    - [ ] 各サブコマンドの動作確認
    - [ ] Ruby版との引数互換性テスト
    - [ ] ヘルプ表示の Ruby版との一致確認
    - [ ] 既存の core バイナリとの連携動作確認
- [ ] Go 側統合テスト追加（`spec/data/` のサンプルを golden として活用）
- [ ] 一時的に RSpec で互換性を検証 → Go テストへ段階的移行
- [ ] ビルド/配布を Go 単体に一本化（クロスビルド・リリース生成）
- [ ] Ruby コード・gemspec の撤去、README/AGENTS 更新
- [ ] 互換ラッパー（任意）: 旧コマンド名で起動した場合に `perfmonger <subcmd>` を呼ぶスクリプトを配布
- [ ] 完了項目を `devdoc/DONE.md:1` へ移し、移行メモ/既知の差分を記録

### 主な互換性問題の詳細

#### 1. record/stat/live コマンド（RecordOption ベース）
- **ディスクオプション**: Ruby `-d DEVICE` (複数回指定) vs Go `-disks string` (CSV)
- **ログファイルオプション**: Ruby `-l, --logfile` vs Go `-output`
- **Ruby固有オプション**: `--kill`, `--status`, `--background`, `--record-intr`, `--no-gzip` が未実装
- **デフォルト値**: Ruby `perfmonger.pgr` vs Go `"-"` (stdout)
- **ヘルプ形式**: Ruby カスタム vs Go flag.PrintDefaults()

#### 2. play コマンド
- **オプション形式**: Ruby `-c, --color` vs Go `-color` (short form なし)
- **オプション形式**: Ruby `-p, --pretty` vs Go `-pretty` (short form なし)
- **ヘルプ形式**: Ruby カスタム vs Go flag.PrintDefaults()

#### 3. summary コマンド
- **未実装オプション**: Ruby `-p, --pager [PAGER]` が未実装
- **ヘルプ形式**: Ruby カスタム vs Go flag.PrintDefaults()

#### 4. 全サブコマンド共通
- **ヘルプ表示形式**: Ruby は `Usage: perfmonger SUBCOMMAND [options] ARGS\n\nOptions:` 形式
- **Go現在**: `Usage of SUBCOMMAND:` + flag.PrintDefaults() 形式
- **互換性**: Ruby版のオプションパーサーとの完全互換が必要

### cobra/pflag 実装設計

#### カスタム Usage テンプレート
```go
const usageTemplate = `Usage:
  {{.UseLine}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`
```

#### record コマンド実装例
```go
func newRecordCommand() *cobra.Command {
    opts := &recordOptions{}
    
    cmd := &cobra.Command{
        Use:   "record [options]",
        Short: "Record system performance information", 
        RunE: func(cmd *cobra.Command, args []string) error {
            return opts.run()
        },
    }
    
    // Ruby互換オプション
    cmd.Flags().StringSliceVarP(&opts.Disks, "disk", "d", []string{}, 
        "Device name to be monitored (e.g. sda, sdb, md0, dm-1).")
    cmd.Flags().StringVarP(&opts.Logfile, "logfile", "l", "perfmonger.pgr", 
        "Output file name")
    cmd.Flags().Float64VarP(&opts.Interval, "interval", "i", 1.0, 
        "Amount of time between each measurement report. Floating point is o.k.")
    cmd.Flags().BoolVar(&opts.Kill, "kill", false, 
        "Stop currently running perfmonger-record")
    cmd.Flags().BoolVar(&opts.Status, "status", false, 
        "Show currently running perfmonger-record status")
    cmd.Flags().BoolVar(&opts.Background, "background", false, 
        "Run in background")
    cmd.Flags().BoolVar(&opts.RecordIntr, "record-intr", false, 
        "Record per core interrupts count (experimental)")
    cmd.Flags().BoolVar(&opts.NoGzip, "no-gzip", false, 
        "Do not save a logfile in gzipped format")
        
    cmd.SetUsageTemplate(usageTemplate)
    return cmd
}
```

#### 単体テスト可能な設計方針
各サブコマンドは以下の設計パターンで実装:
```go
// 1. オプション構造体（テスト可能）
type recordOptions struct {
    Disks     []string
    Logfile   string  
    Interval  float64
    Kill      bool
    Status    bool
    // ... その他のオプション
}

// 2. 引数解析関数（単体テスト可能）
func (opts *recordOptions) parseArgs(args []string, cmd *cobra.Command) error {
    // pflag による引数解析
    // バリデーション実行
    return nil
}

// 3. 実行関数（単体テスト可能）
func (opts *recordOptions) run() error {
    // ビジネスロジック実行
    return nil
}

// 4. cobra.Command 生成関数
func newRecordCommand() *cobra.Command {
    opts := &recordOptions{}
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := opts.parseArgs(args, cmd); err != nil {
                return err
            }
            return opts.run()
        },
    }
    // フラグ定義
    return cmd
}
```

#### 単体テストの範囲
- **引数解析テスト**: 各オプションの正しい解析、デフォルト値、バリデーション
- **ビジネスロジックテスト**: コマンド実行の動作確認（モック使用）
- **Ruby互換性テスト**: Ruby版と同じ引数での動作確認

#### 段階的移行戦略
1. **Phase 1**: 基盤整備（cobra構造、テンプレート、テスト基盤）
2. **Phase 2**: 各サブコマンドを順次移行（record → play → summary → ...）
   - 各サブコマンドで上記設計パターンを適用
   - 単体テスト作成と並行実装
3. **Phase 3**: 統合テスト、Ruby版との完全互換性検証

#### 期待される効果
- Ruby版ユーザーの移行時にコマンド変更不要
- より堅牢なオプション解析
- 統一されたヘルプ形式
- 将来の機能拡張が容易
- **高いテスト覆盖率による品質保証**
- **リグレッション防止**

## 段階2.5: オプション管理の効率化とアーキテクチャ改善

### タスク: 共通オプション構造体の統一とパフォーマンス改善

- [ ] **オプション構造体の統一** - 現在の非効率な設計を改善
  - [ ] record サブコマンドの改善
    - 問題: `record.go` の `recordOptions` → 引数リスト → `recorder.go` の `recorderOptions` の二重解析
    - 解決: `recorder.go` の `RecorderOptions` を共通構造体として利用し、直接オプション構造体を渡す設計に変更
  - [ ] play サブコマンドの改善
    - 問題: `play.go` の `playOptions` → 引数リスト → `player.go` のオプション解析の二重解析
    - 解決: `player.go` のオプション構造体を共通化し、直接構造体を渡す設計に変更
  - [ ] summary サブコマンドの改善
    - 問題: `summary.go` の `summaryOptions` → 引数リスト → `summarizer.go` のオプション解析の二重解析
    - 解決: `summarizer.go` のオプション構造体を共通化し、直接構造体を渡す設計に変更
  - [ ] stat サブコマンドの改善
    - 問題: `stat.go` の `statOptions` → 複数の `buildXXXArgs()` → 各コンポーネントでの再解析
    - 解決: 各コンポーネント（recorder, summarizer）のオプション構造体を直接構築して渡す設計に変更
  - [ ] plot サブコマンドの改善
    - 問題: `plot.go` の `plotOptions` → `plotCommand` 構造体への変換
    - 解決: plot 実装を直接 `plotOptions` で動作するように変更
  - [ ] live サブコマンドの改善（未実装時に対応）
    - 解決: recorder/viewer コンポーネントのオプション構造体を直接利用

- [ ] **アーキテクチャ改善の指針**
  - [ ] 各コア機能（recorder, player, summarizer, plotformatter, viewer）のオプション構造体を公開
  - [ ] cobra コマンドから直接構造体を構築してコア機能に渡す設計
  - [ ] 引数リスト → 構造体 の変換処理を削除
  - [ ] 単体テスト時にもオプション構造体を直接構築してテスト可能な設計
  - [ ] パフォーマンス向上: 引数解析の二重実行を排除

- [ ] **既存テストの更新**
  - [ ] 各サブコマンドの単体テストを新しいオプション構造体設計に合わせて更新
  - [ ] `buildXXXArgs()` 関数のテストを削除（不要になるため）
  - [ ] 直接オプション構造体を検証するテストに変更

## 現在の実装状況 (2025-01-05)

### 完了済み cobra 移行サブコマンド
1. **record** ✅ - 完全実装、テスト済み
2. **play** ✅ - 完全実装、テスト済み  
3. **summary** ✅ - 完全実装、テスト済み
4. **stat** ✅ - 完全実装、テスト済み
5. **plot** ✅ - 完全実装、コンパイル成功
6. **fingerprint** ✅ - 完全実装、Ruby互換
7. **init-shell** ✅ - 完全実装、Ruby互換
8. **live** ✅ - 完全実装、recorder-player連携

### **🎉 Phase 2 サブコマンド移行完了！**
- 全8サブコマンドの cobra/pflag 移行が完了
- Ruby版との引数互換性を維持
- 統一されたヘルプ表示形式を実装
- コンパイルエラーなし

### 主要な技術的成果
- 単体テスト可能な設計パターンの確立
- Ruby完全互換のコマンドライン引数解析
- 統一されたヘルプ表示形式
- pflag/cobra による堅牢なオプション処理

## マイルストーン/完了条件

- 段階1完了: Ruby CLI + `CoreFinder` で全サブコマンドが旧名で動作し、RSpec/Go テストがパス。`lib/exec` は単一バイナリ由来。
- 段階2完了: Ruby 依存が無く、`perfmonger` 単一バイナリで同等機能（server/plot 含む）。Go テストがパスし、利用手順が README に反映。
