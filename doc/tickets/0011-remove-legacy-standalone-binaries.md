---
title: Remove legacy standalone perfmonger-core binaries and their build rules
type: refactor
priority: medium
status: open
created: 2026-05-29
updated: 2026-05-29
---

## Description

Go 単一バイナリ化により、ユーザ向け CLI は `perfmonger`（cobra）1 本で全サブ
コマンドを **in-process** で実行するようになった。`perfmonger` は core パッケージ
（`recorder`/`player`/`summarizer`/`plotformatter`）を直接 import して呼び出し
（live.go:85, play.go:51, summary.go:96, stat.go:141/173, plot.go:268）、
`--background` も自分自身を re-exec する（record.go:360）。

その結果、かつてサブプロセスとして呼ばれていた **standalone 実行ファイル群が不要**
になっている。本チケットでは、これら不要バイナリと生成ルールが残っていないか
洗い出し、削除・無効化する。

このチケットは #0007（リリースパイプライン）の D2 確定時に分離されたもの。
#0007 のリリースには `perfmonger` 単体のみを同梱するため、legacy バイナリの
除去はリリースの前提ではないが、ビルド・配布の保守性のために整理する。

### 削除/無効化の候補

不要（実行時に誰からも呼ばれない standalone main）:

- `core/cmd/perfmonger-core/perfmonger-core.go` — multi-call ディスパッチャ本体
- `core/cmd/perfmonger-recorder/`
- `core/cmd/perfmonger-player/`
- `core/cmd/perfmonger-summarizer/`
- `core/cmd/perfmonger-plot-formatter/`
- `core/cmd/perfmonger-viewer/`
- [Makefile](Makefile) の `CORE_BIN` / `wrappers` ターゲット / `CORE_SUBCMDS` 変数、
  および `build` ターゲットの `$(CORE_BIN)` 依存・`clean` の該当行

### 必ず残すもの（削除してはならない）

`perfmonger` が import している **再利用パッケージ**。これらは
`core/cmd/perfmonger-core/` 配下のサブディレクトリにあり、ディスパッチャ本体
（`perfmonger-core.go`）とは別物である:

- `core/cmd/perfmonger-core/recorder/`
- `core/cmd/perfmonger-core/player/`
- `core/cmd/perfmonger-core/summarizer/`
- `core/cmd/perfmonger-core/plotformatter/`
- `core/cmd/perfmonger-core/viewer/`（import 状況を要確認。未 import でも他から
  参照がないか確認のうえで判断）

注: ディスパッチャ本体 `perfmonger-core.go` はパッケージ群と同じ
`core/cmd/perfmonger-core/` ディレクトリに同居しているため、**ファイル単位で
削除**する（ディレクトリごと消すとパッケージまで失う）。

### 作業前の確認事項

- 上記 standalone cmd を参照しているもの（`build.sh` は既に存在しない／
  devcontainer・CI・テスト・ドキュメント・[doc/architecture.md](doc/architecture.md)）が
  ないか grep で洗い出す。
- 互換シンボリックリンク（`perfmonger-recorder_*` 等）に依存する外部スクリプトや
  pytest フィクスチャ（[tests/conftest.py](tests/conftest.py)）が無いか確認。
- `viewer` パッケージが `perfmonger` から（間接的にでも）使われているか確認。

### 受け入れ条件

- `make build` / `make cross-build` が `perfmonger` のみを生成し成功する。
- `make test`（Go unit + pytest）が緑のまま。
- `go vet` が通る。
- ドキュメント（[doc/architecture.md](doc/architecture.md) のリポジトリレイアウト節など）が
  削除後の構成と一致するよう更新される。
