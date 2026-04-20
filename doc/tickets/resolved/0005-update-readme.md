---
title: Rewrite README.md for Go single-binary distribution
type: docs
priority: medium
status: resolved
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

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Implementation Notes 内の Resolution で全8判断事項が確定済み（インストールは両論併記、チュートリアル寄り、サブコマンドは record/play/stat/summary/plot を簡潔紹介、出力例は刷新、gnuplot 依存は軽く言及、テスト手段と開発者向け記述は README から外し CLAUDE.md に寄せる、バッジ撤去）。残るのはプロセ生成とサンプル差し替えで、設計判断は無い。

## Implementation Notes

### 判断が必要な項目

1. **インストール記述のスコープ**: (a) GitHub Releases からのバイナリダウンロードだけ / (b) ソースから `go build` / (c) 両方併記。#0007 のリリース基盤が未完なので、暫定的に (b) で案内するのか、"Coming soon" で (a) の枠だけ用意するのか。
2. **構成とトーン**: 既存の "Target platform → Installation → Getting started → Typical use cases" を踏襲するか、"Quickstart / Subcommands / Testing / Development" に再編するか。チュートリアル寄りか簡潔なリファレンス寄りか。
3. **サブコマンド網羅度**: 全8サブコマンド（record, play, summary, plot, live, stat, fingerprint, init-shell）を一行ずつ列挙するか、record/play/live にフォーカスして残りは `perfmonger --help` 参照にするか。
4. **旧例・出力サンプルの扱い**: 現行 README の JSON 出力サンプルと `-i 0.1 -d sda` 等のフラグ例を維持するか、刷新するか。
5. **外部依存の明記**: `plot` サブコマンドに必要な gnuplot、`/proc/diskstats` 依存、権限要件などをどの深さで書くか。
6. **テストセクション配置**: `uv sync && uv run pytest -v` を Getting Started 側に置くか独立した Testing セクションに分けるか。
7. **開発者向け記述**: "Go Dev Environment (optional)" 節を README に残すか、[CLAUDE.md](CLAUDE.md) / devcontainer ドキュメントに寄せるか。
8. **バッジとレガシー記述**: 旧 Travis / Wercker バッジは撤去、Ruby/Go 二段構成時代への言及は完全削除で良いか。GitHub Actions バッジは #0007 完了後に追加するか。

決めるべき順序: まず 1〜2（全体像）→ 3〜4（本文の深さ）→ 5〜8（付随情報）。

## Decisions (User)

- インストール記述 (c)両方
- 構成とトーン チュートリアル寄り
- サブコマンド網羅度 record/play/stat/summary/plot を簡単に紹介
- 出力サンプルは刷新
- 外部依存は依存コマンドgnuplotの説明を簡単に
- テスト手段は CLAUDE.md のみに記載していればOKで、README.mdにはいらない
- 開発者向け記述は README から撤去して CLAUDE.md に寄せる
- バッジは削除

## Resolution

Rewrote `README.md` end-to-end following the 8 user decisions above.

**Sections removed (from old README):**
- Travis and Wercker badges.
- `gem install perfmonger` and Ruby/Go two-binary build instructions.
- `Go Dev Environment (optional)` block (`source 00_LOAD_GO_DEVENV.sh`) —
  developer-facing details now live in `CLAUDE.md`.
- No testing section in README (pytest instructions remain only in
  `CLAUDE.md`).

**Sections kept or newly added:**
- Short intro, feature bullets, and Linux-only platform note.
- Installation: "Pre-built binary (coming soon)" placeholder referencing the
  release pipeline tracked in #0007, plus a primary `go build` from-source
  path (Go 1.24+ to match `core/go.mod`).
- Optional-dependencies line mentioning that `gnuplot` is only needed for
  `perfmonger plot`.
- Quick tour with a short intro per subcommand for the five chosen
  (`record`, `play`, `stat`, `summary`, `plot`), each backed by a freshly
  captured output sample. Note about `--background` / `--status` / `--kill`
  being specific to `record`. `perfmonger --help` is shown for the full
  subcommand list.
- Condensed JSON record schema reference at the end.

**Output samples:** Freshly generated against the current binary
`/workspaces/perfmonger/lib/exec/perfmonger_linux_amd64` (version 0.14.4) on
this environment:
- `perfmonger --help`
- `perfmonger record -i 0.5 --timeout 3 -d nvme0n1 -l /tmp/sample.pgr.gz`
- `perfmonger play /tmp/sample.pgr.gz | head -n 1`
- `perfmonger stat -d nvme0n1 -- sh -c '...'`
- `perfmonger summary --disk-only '^nvme0n1$' /tmp/sample.pgr.gz`

**Verification:**
- Confirmed `go build` snippet in README builds the binary successfully.
- Confirmed all snippet commands executed without error (except the
  deliberately mentioned gnuplot dependency for `plot`).
- Ran `uv sync && uv run pytest -v`: 25 passed, 4 skipped, 7 failed. The 7
  failures are all in `tests/test_background.py` and reproduce on the
  unchanged tree (pre-existing, unrelated to the README change).