---
title: Remove leftover Ruby sources and gem packaging
type: chore
priority: high
status: resolved
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

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: 削除対象と保持対象が本チケット本文で明確に分離されており、`spec/data/` は pytest の `conftest.py` から `DATA_DIR` 経由で参照されているので削除禁止。`Rakefile` は #0006 が所有するスコープなので本チケットでは手を付けない。アクティブなコードから Ruby ファイルへの参照は無く（`core/build.sh` / `core/Makefile` に Ruby 参照なし）、削除後に pytest がグリーンであれば完了。`.gitignore` の `/Gemfile.lock`・`/perfmonger-*.gem` エントリも不要になるので合わせて削除する。

## Resolution

Deleted Ruby-era assets via `git rm -r`:

- `lib/perfmonger.rb` and entire `lib/perfmonger/` tree (CLI + `command/` subdirectory)
- `exe/perfmonger` (and `exe/` directory removed automatically once empty)
- `Gemfile`, `perfmonger.gemspec`
- All RSpec tests under `spec/` (`*_spec.rb`, `spec_helper.rb`, `spec/support/aruba.rb`)

Also removed untracked Ruby artifacts on disk: `Gemfile.lock` and `pkg/perfmonger-0.14.4.gem` (plus the now-empty `pkg/` directory).

`.gitignore` cleanup: removed the `/perfmonger-*.gem`, `/pkg`, `/Gemfile.lock`, and `/spec/examples.txt` entries which only existed to mask Ruby toolchain artifacts.

Preserved as instructed: `lib/exec/` (Go binaries + compatibility symlinks) and `spec/data/` (pytest golden fixtures). `Rakefile` left in place for ticket #0006.

Verification:
- `cd core/internal/perfmonger && go test -v -cover`: PASS (15 tests, 54.2% coverage).
- `cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)`: clean.
- `uv run pytest -v`: 32 passed, 4 skipped (gnuplot unavailable). No regressions attributable to this change.

Follow-ups (not in this ticket's scope):
- `AGENTS.md` still references deleted Ruby paths; tracked by ticket #0005 (README rewrite).
- `Rakefile` removal is ticket #0006.
