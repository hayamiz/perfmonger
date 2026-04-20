---
title: Remove leftover Ruby sources and gem packaging
type: chore
priority: high
status: open
created: 2026-04-20
updated: 2026-04-20
---

## Description

Go 単一バイナリへの移行は完了済み（[doc/DONE.md](doc/DONE.md) 段階2）だが、Ruby 実装のファイル群が一切撤去されていない。現在リポジトリに残っているもの:

- `lib/perfmonger/` — Ruby CLI 実装（`cli.rb`, `command/`, `config.rb`, `version.rb`）
- `exe/perfmonger` — Ruby エントリポイント
- `Gemfile`, `Gemfile.lock` — Bundler 依存管理
- `perfmonger.gemspec` — gem パッケージ定義
- `spec/*.rb` — RSpec テスト9ファイル（`fingerprint_spec.rb` など。pytest に全面置換済み）
- `Rakefile` — Ruby ベースのビルド/テストタスク（※ Rakefile 自体の扱いは [0006-replace-rakefile.md](0006-replace-rakefile.md) で扱う）

**保持すべきもの（誤削除注意）:**
- `lib/exec/` — Go バイナリ (`perfmonger_linux_amd64` 等)、互換シンボリックリンクの置き場所
- `spec/data/` — pytest のゴールデンフィクスチャ（`.pgr`, `.pgr.gz`）として `tests/conftest.py` から参照中

やること:
- 上記 Ruby 資産を削除（`git rm -r`）
- `lib/` 直下は `exec/` のみを残す形に整理
- `.gitignore` から不要な Ruby 関連エントリを削除
- pytest 統合テストがグリーンで完走することで機能回帰が無いことを確認

関連チケット: [0005-update-readme.md](0005-update-readme.md), [0006-replace-rakefile.md](0006-replace-rakefile.md)
