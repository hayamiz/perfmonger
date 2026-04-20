---
title: Set up cross-build and release pipeline
type: feature
priority: low
status: open
created: 2026-04-20
updated: 2026-04-20
---

## Description

現在のリポジトリには自動化されたリリース機構が無い:
- `.github/workflows/` は存在しない
- `goreleaser` 設定は無い
- `release.sh` の類も無い
- クロスビルドは `core/build.sh` が手動実行されるのみ

Go 単一バイナリ化が完了したので、ユーザーが配布物を入手できる経路を整える。

選択肢（設計判断が必要）:
1. **GoReleaser + GitHub Actions** — タグ push で tarball/zip を release に添付。標準的で軽量。
2. **手書き GitHub Actions workflow** — `go build` をマトリクスで回して artifact を release に添付。依存最小。
3. **ローカル `make release` + 手動 upload** — CI を使わない。保守最小だが属人化。

現時点の方針は未定。`/ticket-triage` 実行時にユーザー判断を仰ぐべき項目。

最低限のスコープ（方針決定後）:
- Linux amd64 / arm64 バイナリを生成
- タグ（例: `v1.x.y`）に紐づくリリース成果物として公開
- SHA256 チェックサム同梱
- README ([0005-update-readme.md](0005-update-readme.md)) のインストール手順からダウンロード URL にリンク可能にする

依存順序: [0004-remove-ruby-code.md](0004-remove-ruby-code.md) → [0006-replace-rakefile.md](0006-replace-rakefile.md) → 本チケット。

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: アプローチの選択（GoReleaser / 手書き Actions / ローカル Make）自体がユーザ判断。さらにバージョン管理方針（現状 `main.go` の const と `lib/perfmonger/version.rb` の二重管理）、配布対象アーキ、チェックサム以上の署名の要否、Homebrew tap / Docker image 等の追加チャンネルの要否も決める必要がある。

## Implementation Notes

### 3案の比較

| 観点 | 1. GoReleaser + GH Actions | 2. 手書き GH Actions | 3. ローカル `make release` |
|---|---|---|---|
| 学習コスト | `.goreleaser.yaml` スキーマ | YAML actions のみ | 最小（Makefile のみ） |
| 依存増 | goreleaser CLI | なし | なし |
| 自動化レベル | タグ push → 配布物生成・アップロードまで完全自動 | 同左（手書きなので作り込み次第） | 手動アップロード必要 |
| 将来の拡張 | Homebrew tap / SBOM / 署名がフラグ一つ | 都度実装 | 都度実装 |
| 監査性 | CI ログ | CI ログ | 人のマシンに依存 |

**推奨: Option 1 (GoReleaser)**。単一バイナリ + 成果物が tar.gz + SHA256 という素直な構成なので GoReleaser の機能過多は問題にならず、将来の配布チャンネル追加（Homebrew / Docker）が宣言的に済む。

### 追加で要判断の項目

1. **バージョン同期**: 現状 [core/cmd/perfmonger/main.go](core/cmd/perfmonger/main.go) と `lib/perfmonger/version.rb` で `0.14.4` が二重管理。Single source of truth 方針は (a) Go const を維持しタグ時に `-ldflags "-X main.VERSION=<tag>"` で注入 / (b) 現状維持で手動同期 / (c) `version.rb` は #0004 で消えるので自然に (a) になる、のどれか。
2. **対象プラットフォーム**: ticket は linux amd64 + arm64。linux 386 / freebsd / macOS（Darwin は削除済み）を追加する予定はあるか。
3. **アーカイブ形式**: `.tar.gz` のみで良いか（Linux のみなら十分）。同梱ファイル（LICENSE, README, NEWS）の取捨。
4. **署名**: SHA256 チェックサムに加えて GPG 署名 / cosign 等を要するか。
5. **リリースノート**: goreleaser 自動生成 / 手書き / `NEWS` ファイルの該当節をコピー、のどれか。
6. **Homebrew tap / Docker image / SBOM**: 本チケットのスコープ外とするか、0007 続編として follow-up チケット化するか。
7. **CI のテストゲート**: リリース前に `make test` を走らせるか。走らせるとすれば GH runner で `/proc/diskstats` 依存テストをどう扱うか（`@requires_proc_diskstats` スキップで足りる）。

### #0006 との棲み分け

`make cross-build` が linux/amd64 + linux/arm64 を出力する前提なら、**Option 2 (手書き Actions)** は単に `make cross-build` を呼ぶだけで済む（コード重複無し）。**Option 1 (GoReleaser)** は自前のマトリクスを持つため `make cross-build` とは二重になるが、用途が違う（ローカル開発ビルド vs リリースアーティファクト）ので共存可。どちらに寄せるかは 1 で GoReleaser を選ぶなら `make cross-build` は開発者便宜のみとし、CI は GoReleaser 側に任せるのが自然。

### 方針決定後の具体スコープ（Option 1 採用を仮定）

追加ファイル:
- `.goreleaser.yaml` — builds / archives / checksum / release 設定、linux amd64+arm64
- `.github/workflows/release.yml` — `v*.*.*` タグ push で `goreleaser release --clean` を実行
- `.github/workflows/ci.yml` — push/PR で `make test` 相当を実行（任意だが推奨）

README (#0005) への反映:
- Installation セクションから GitHub Releases ページへリンク
- `curl -L ... | tar xz` の例示を追記
