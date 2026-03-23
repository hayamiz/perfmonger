# DONE: perfmonger 完了済みタスク

## 段階0: 不要な機能の整理、コード構造の整理

- [x] Darwin サポートの廃止
- [x] パッケージ構造のリファクタリング（`core/subsystem` → `core/internal/perfmonger`）

## 段階1: core を単一バイナリ化（`perfmonger-core`）

- [x] `core/cmd/perfmonger-core` で record/play/summarize/plot-format/view サブコマンドを統合
- [x] 各コンポーネントを `Run(args)` 公開関数としてパッケージ化
- [x] `lib/exec/` に互換シンボリックリンク配置

## 段階2: Ruby を廃止し Go へ完全移行（単一 `perfmonger`）

- [x] Go による `perfmonger` 単一バイナリで全サブコマンド実装
  - live, record, play, summary, plot, stat, fingerprint, init-shell
  - server サブコマンドは廃止
- [x] cobra/pflag への移行と Ruby 版互換性修正（全8サブコマンド完了）
- [x] Ruby 互換セッション管理（record コマンドの --background / --kill / --status）

## 段階2.5–2.8: アーキテクチャ改善

- [x] オプション構造体の統一（RecorderOption, PlayerOption, SummaryOption 公開化）
- [x] cobra フラグからオプション構造体への直接バインディング
- [x] 二重引数解析（cobra → args → parseArgs）の排除
- [x] recordCommand 等の直接設定方式への移行（全サブコマンド）
- [x] recorder の RunDirect API 実装
- [x] plotformatter の RunDirect API 実装（PlotFormatOption 公開構造体）

## バグ修正

- [x] record --background: fork() を re-exec パターンに置換（Go ランタイムは fork 非対応）、出力パスの絶対パス変換
- [x] record --no-gzip: デフォルトファイル名から `.gz` サフィックスを自動除去
- [x] live コマンド: player binary を `os.Executable() + "play"` に変更、nil pointer ガード追加
- [x] stat コマンド: StopCh チャネルで recorder を明示的に停止し unexpected EOF を解消
- [x] plot コマンド: plotformatter.RunDirect() の直接呼び出しに変更、gnuplot スクリプト生成を完全実装

## テスト基盤

- [x] pytest 統合テスト基盤の構築（RSpec/Aruba を置換）
  - uv + pytest + pytest-timeout
  - `tests/` 以下に9ファイル、36テスト
  - `spec/data/` のゴールデンファイルを活用
- [x] 段階3（Bats テスト基盤）は pytest で代替・完了
