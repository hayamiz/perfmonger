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
