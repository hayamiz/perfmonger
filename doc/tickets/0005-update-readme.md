---
title: Rewrite README.md for Go single-binary distribution
type: docs
priority: medium
status: open
created: 2026-04-20
updated: 2026-04-20
---

## Description

`README.md` が Ruby gem 時代の前提で書かれたままになっている。具体的には:

- `gem install perfmonger` を使ったインストール手順が残存
- Ruby 2.2+ と Go 1.8+ の両方をビルド要件として記載
- Ruby CLI と Go コアの二段構成を前提にした説明

現在のプロジェクトは Go 単一バイナリ（`lib/exec/perfmonger_linux_amd64`）で完結しており、Ruby 依存は [0004-remove-ruby-code.md](0004-remove-ruby-code.md) で撤去予定。README もそれに合わせて書き直す。

反映すべき内容:
- Linux 専用（Darwin サポートは [doc/DONE.md](doc/DONE.md) 段階0で廃止済み）
- インストール手順は Go バイナリのダウンロード（リリース生成が整ったら [0007-release-pipeline.md](0007-release-pipeline.md) のアーティファクトに差し替え可能なよう暫定案内）またはソースから `go build`
- 現行のサブコマンド一覧（record, play, summary, plot, live, stat, fingerprint, init-shell）
- `--background` / `--status` / `--kill` は record 専用（live/stat からは削除済み）
- pytest 統合テストの実行方法（`uv sync && uv run pytest -v`）

順序としては [0004-remove-ruby-code.md](0004-remove-ruby-code.md) 完了後が望ましい（削除対象のファイルを README が参照し続けないように）。
