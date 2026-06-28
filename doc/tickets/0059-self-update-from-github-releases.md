---
title: GitHub Releases を使った自己アップデート機能 (perfmonger --update)
type: feature
priority: medium
status: open
created: 2026-06-27
updated: 2026-06-27
---

## Description

`perfmonger` 自身が GitHub Releases を参照して、より新しいリリースがあれば
実行中のバイナリを最新版へ自己更新できるようにする。

### 動作イメージ

```
perfmonger --update
```

1. GitHub Releases (`github.com/hayamiz/perfmonger`) の最新リリースを取得し、
   現在のバージョン（`var VERSION`、リリースビルドでは `-ldflags -X main.VERSION`
   でタグ値が注入される）と比較する。
2. 新しいリリースが存在する場合、現在のバージョンから最新リリースまでの
   **リリースノート一覧**を表示し、ユーザーに更新するか確認する。
3. ユーザーが更新に同意した場合、実行中のアーキテクチャ（linux amd64 / arm64）に
   対応した成果物をダウンロードし、現在の実行バイナリを置き換える。

### 既存の前提（[0007](resolved/0007-release-pipeline.md) で確定済み）

- リリースは GoReleaser + GitHub Actions（`v*.*.*` タグ push）で自動生成。
- 配布物は **`perfmonger` 単体**を含む arch 別 tar.gz（linux amd64 / arm64）と、
  ソース tar.gz、`checksums.txt`（SHA256）。
- バージョンは `core/cmd/perfmonger/main.go` の `var VERSION = "dev"` が
  single source of truth。ローカルビルドは `dev`、リリースのみタグ値が注入される。
- リリースノートは `NEWS.md` の該当バージョン節をキュレートして
  `--release-notes` で渡している（git-log 自動生成は無効）。

### 検討が必要な設計判断（triage / grill で詰める）

- **バージョン比較**: `dev` ビルド（ローカル開発版）に対して `--update` を
  どう振る舞わせるか（拒否 / 警告して続行など）。semver 比較ロジックの導入要否。
- **取得経路**: GitHub Releases API（`/repos/hayamiz/perfmonger/releases`）を
  叩くか、`latest` のリダイレクトを使うか。認証なし（公開リポジトリ）で
  レート制限が問題にならないか。
- **アーキテクチャ判定**: `runtime.GOARCH` から成果物名（GoReleaser の
  アーカイブ命名規則）へのマッピング。amd64 / arm64 以外（未配布 arch）の扱い。
- **検証**: ダウンロードした tar.gz を `checksums.txt`（SHA256）で必ず検証する。
  GPG/cosign 署名は現状未導入（[0007](resolved/0007-release-pipeline.md) で
  follow-up 扱い）なので SHA256 検証までをスコープとするか。
- **バイナリ置き換え**: 実行中バイナリの自己書き換え方法。多くの OS で
  実行中ファイルの上書きは不可なため、同一ディレクトリへ一時ファイルとして
  展開 → `os.Rename`（atomic replace）→ パーミッション保持、の手順が必要。
  書き込み権限が無い場合（system-wide インストール）のエラーハンドリングと
  ユーザー向けメッセージ（要 sudo など）。
- **CLI 形態**: `--update` をルートフラグにするか `perfmonger update` サブ
  コマンドにするか。`--check`（更新の有無だけ確認）、`--yes`（確認スキップ）、
  バージョン指定での pin/downgrade を提供するか。
- **依存追加の是非**: 自己更新ライブラリ（例: `rhysd/go-github-selfupdate`,
  `creativeprojects/go-selfupdate`, `minio/selfupdate` 等）を使うか自前実装か。
  追加依存の最小化方針との兼ね合い。
- **リリースノート一覧の表示**: 現在〜最新の間に複数リリースがある場合、
  各リリースの body を順に表示する。pager（[0001](resolved/0001-enable-pager-in-summary.md)
  で導入済みの仕組み）を流用できるか。

### 関連

- リリースパイプライン: [0007](resolved/0007-release-pipeline.md)
- pager 機構: [0001](resolved/0001-enable-pager-in-summary.md)
