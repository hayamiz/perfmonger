---
title: Set up cross-build and release pipeline
type: feature
priority: medium
status: open
created: 2026-04-20
updated: 2026-05-29
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

_(re-triaged 2026-05-29 — 主要な設計判断は `## Decisions` で確定済み。本セクションはその後の再分析)_

- Complexity: medium
- Mechanical fix: yes _(grill 後。D1–D3 と運用設計が確定し、残りは機械的)_
- Requires user decision: no _(grill で D1–D3 を解決済み)_
- Notes: アプローチ（GoReleaser）・バージョン（v1.0.0）・対象アーキ（linux amd64/arm64）・
  成果物範囲は `## Decisions` で確定。2026-05-29 のグリル（`/grill-with-ticket`）で
  残っていた要判断 D1（NEWS→NEWS.md 改名＋1.0.0 エントリ書き換え）・D2（配布は
  `perfmonger` 単体、legacy バイナリ除去は #0011 に分離）・D3（リリースノートは
  NEWS.md 該当節を `--release-notes` で渡す）と、運用設計（`var VERSION="dev"`、
  ci.yml/release.yml の構成・テストゲート）をすべて確定した。技術前提（const→var、
  module root `core/`、goreleaser v2、COPYING=GPLv3、cgo-free クロスビルド、
  `make build`→`make test` 順、runner に uv）も検証・記録済み。詳細は下記
  「### Triage findings」「### 運用設計の確定」「### 実装上の注意」を参照。実装は
  C〜G を機械的に適用すればよい。

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

## Decisions (2026-05-29)

「Go 版に置き換えた最初の正式リリース」を当面のゴールとして、トリアージ時の
未決事項を以下のとおり確定した。

- **A. バージョン**: 初回タグは **`v1.0.0`**（Go 単一バイナリ化の節目を 1.0 と
  して切る）。実装は `core/cmd/perfmonger/main.go` の `const VERSION` を
  single source of truth とし、リリースビルド時に
  `-ldflags "-X main.VERSION=<tag>"` でタグ値を注入する。`version.rb` は
  #0004 で削除済みなので二重管理は解消済み。
- **B. 対象プラットフォーム**: **linux/amd64 + linux/arm64** のみ。コードは
  Linux 専用（`perfmonger_linux.go`）のため 386 / macOS は初回スコープ外。
- **H. 成果物**: 各アーキ向けの **ビルド済みバイナリ**、**ソースコードの
  tar.gz**、および **SHA256 チェックサム**。GPG/cosign 署名・Homebrew tap・
  Docker image・SBOM は初回では行わず、必要になった時点で follow-up チケット化。
- **採用方式**: Option 1（**GoReleaser + GitHub Actions**）。

### 確定スコープ（実装タスク）

| # | タスク | 種別 |
|---|---|---|
| C | `.goreleaser.yaml`（builds: linux amd64/arm64, archives: tar.gz, checksum: SHA256, release, ldflags でバージョン注入） | 実装 |
| D | `.github/workflows/release.yml`（`v*.*.*` タグ push → `goreleaser release --clean`） | 実装 |
| E | `.github/workflows/ci.yml`（push/PR で `make test`。`/proc/diskstats` 依存テストは `@requires_proc_diskstats` でスキップ） | 実装（推奨） |
| F | README の Installation を実手順へ（Releases リンク + `curl -L ... \| tar xz` 例） | docs |
| G | **`RELEASING.md`（メンテナ向けリリース手順書）の新規作成** — タグの切り方 → CI 発火 → 成果物確認 → README 更新までの一連手順 | docs |

注: G はトリアージ時の実装ノートで明示タスク化されていなかったが、
「ユーザー（メンテナ）が読みやすいリリース手順文書」が現状ゼロのため本チケットに含める。

### Triage findings (2026-05-29 — 実装着手前に潰すべき技術的事項)

コードベースを再調査した結果、確定スコープ C〜G を機械的に進めるうえで効いてくる
具体的な事実・ブロッカーを以下に列挙する。

1. **`const VERSION` は `-ldflags -X` で上書きできない（要コード変更）。**
   [core/cmd/perfmonger/main.go](core/cmd/perfmonger/main.go) 9 行目は
   `const VERSION = "0.14.4"`。`-ldflags "-X main.VERSION=<tag>"` は
   **パッケージ変数 (`var`) にしか効かず `const` には効かない**。タスク C の前提
   （ldflags 注入）を成立させるには `var VERSION = "0.14.4"`（または
   `var VERSION = "dev"`）へ変更する 1 行修正が必須。`## Decisions` の方式は
   この修正込みで初めて成立する。

2. **Go module root は `core/`。goreleaser の `dir:`/`main:` 指定が必須。**
   [core/go.mod](core/go.mod) は `module github.com/hayamiz/perfmonger/core` /
   `go 1.24.0`。リポジトリルートに `go.mod` は無い（ルート `go.mod` は gitignore 済）。
   goreleaser のデフォルトはルートに `go.mod` がある前提なので、`.goreleaser.yaml`
   の各 `builds:` stanza に `dir: core` を指定し、`main:` はその相対で
   `./cmd/perfmonger`（および必要なら `./cmd/perfmonger-core`）を指す必要がある。
   `actions/setup-go` の `go-version` は `'1.24'` に合わせる。

3. **goreleaser は v2 スキーマを使う。**
   `.goreleaser.yaml` 先頭に `version: 2`。CI は
   `goreleaser/goreleaser-action@v6` + goreleaser `~> v2` をピン推奨。

4. **配布バイナリは 2 種類あるが `perfmonger` 単体で完結する（D2 で確定済み）。**
   ビルドは arch ごとに `perfmonger`（ユーザ向け cobra CLI）と
   `perfmonger-core`（multi-call ディスパッチャ）の 2 バイナリを生成するが、
   `perfmonger` は core パッケージを直接 import して in-process 実行するため
   `perfmonger-core` を実行時に必要としない（D2 参照）。よって配布は
   `perfmonger` のみ。legacy バイナリの除去は #0011 に分離。

5. **ライセンスは `COPYING`（GPLv3）で既存 — 選定不要。**
   ルートに `COPYING`（GNU GPL v3 全文）と `data/NOTICE` あり。`LICENSE`/`LICENSE.md`
   というファイル名ではないため、goreleaser のデフォルト同梱パターンには
   引っかからない。`archives.files` に `COPYING`・`README.md`・`NEWS.md`（D1 で
   `NEWS` から改名）を明示列挙する。

6. **NEWS の次版見出しが `0.15.0` のまま → v1.0.0 と不整合（D1 で確定済み）。**
   ルートの `NEWS` 先頭は `## 2025-XX-XX: PerfMonger 0.15.0`（"Dropped support of
   Darwin" のみ）。D1 のとおり、`NEWS` を `NEWS.md` に改名し先頭エントリを 1.0.0
   用に書き換える。リリースノート源は D3 で確定。

7. **cgo 不使用 → 単一 runner でクロスコンパイル可（検証済み 2026-05-29）。**
   `core/` 配下に `import "C"` は無く、`CGO_ENABLED=0 GOOS=linux GOARCH=arm64
   go build ./cmd/perfmonger` が amd64 上で成功し ARM aarch64 の静的 ELF を生成
   できることを確認済み。よって goreleaser の `builds.env` に `CGO_ENABLED=0` を
   指定すれば、QEMU やマルチ runner 無しで amd64/arm64 両方をビルドできる。

8. **CI で `make test` を回すには runner に `uv` が要る。**
   [Makefile](Makefile) の `test` ターゲットは `uv sync && uv run pytest -v` を含む。
   `ubuntu-latest` には `uv` が未インストールなので、`ci.yml` に
   `astral-sh/setup-uv` ステップを前置する。Go 単体テスト
   （`cd core/internal/perfmonger && go test`）は `uv` 不要なのでステップ分割も可。
   `/proc/diskstats` 依存テストは `@requires_proc_diskstats` でスキップされる想定。

### 実装前に確定すべき小規模な要判断（D1–D3）

- **D1** ✅ **確定（2026-05-29）**: NEWS の先頭 `0.15.0` エントリを **1.0.0 用に書き換える**
  （新規見出しを足すのではなく既存 0.15.0 エントリを置換）。さらに **`NEWS` ファイルを
  `NEWS.md` にリネーム**する。リネームに伴い、アーカイブ同梱ファイル一覧と
  リリースノート源の参照は `NEWS.md` を指すこと（finding #5・#6、D3 参照）。
  `/ticket-fix` で適用する具体内容:
  - `git mv NEWS NEWS.md`
  - 先頭エントリを以下に置換（リリース日は実リリース時に確定）:
    ```
    ## 2026-05-29: PerfMonger 1.0.0
      * Changes
        * Reimplemented entirely in Go and distributed as a single `perfmonger` binary.
        * Removed the Ruby implementation and gem packaging (#0004).
        * Dropped support of Darwin (Mac OS X) which was already deprecated.
        * Started distributing prebuilt Linux (amd64/arm64) binaries via GitHub Releases (#0007).
    ```
  注: 0.15.0 は単独ではリリースされず 1.0.0 に統合される。連続性の注記は付けず、
  見出しを 1.0.0 に置き換えるのみとする（ユーザ確定）。
- **D2** ✅ **確定（2026-05-29）**: リリースアーカイブには **`perfmonger` 単体のみ**を含める。
  根拠: `perfmonger` は `recorder`/`player`/`summarizer`/`plotformatter` パッケージを
  直接 import して in-process で実行し（live.go:85, play.go:51, summary.go:96,
  stat.go:141/173, plot.go:268）、`--background` も自分自身を re-exec する
  （record.go:360）。`perfmonger-core` ディスパッチャや互換シンボリックリンクは
  実行時に不要。よって `.goreleaser.yaml` の `builds:` は `perfmonger` のみ。
  なお、不要になった standalone バイナリ（`perfmonger-core` ディスパッチャ＋
  `perfmonger-{recorder,player,summarizer,plot-formatter,viewer}` cmd 群）と
  Makefile の生成ルールの削除は本チケットのスコープ外とし、**#0011 に分離**。
- **D3** ✅ **確定（2026-05-29）**: リリースノートは **キュレートした `NEWS.md` の該当
  バージョン節を使う**。`release.yml` で `NEWS.md` 先頭エントリ（先頭の `## ` 見出しから
  次の `## ` 直前まで）を awk で抽出し一時ファイルへ出力、`goreleaser release
  --release-notes=<file>` で渡す。goreleaser の git-log 自動生成は無効化する
  （v2 では `changelog: disable: true`）。`NEWS.md` を single source of truth とし、
  リリースノートとの二重管理を避ける。抽出例:
  `awk '/^## /{n++} n==1{print} n==2{exit}' NEWS.md > /tmp/relnotes.md`

D1・D2・D3 はすべて確定済み。これで `/ticket-fix` 着手に必要な設計判断は出揃った。

### 運用設計の確定（2026-05-29 グリル）

- **VERSION 変数のデフォルト値**: finding #1 の `var` 化に際し、デフォルトは
  `var VERSION = "dev"` とする。ローカル `make build` では `perfmonger version` が
  `dev` を表示し、リリースビルドでのみ goreleaser が `-ldflags -X main.VERSION=<tag>`
  でタグ値（先頭 `v` を除いた `1.0.0` 等）を注入する。
- **E: `.github/workflows/ci.yml`** — `push`(master) と `pull_request` で発火。
  `astral-sh/setup-uv` + `actions/setup-go`(`go-version: '1.24'`) を入れ、
  `make test`（Go unit + pytest）と `make vet` を実行。`/proc/diskstats` 依存テストは
  GH runner で `@requires_proc_diskstats` により自動スキップされる。
- **D: `.github/workflows/release.yml`** — `v*.*.*` タグ push で発火。
  **goreleaser 実行前に `make test` をゲートとして実行**し、リリース対象コミットの
  健全性を保証してから `goreleaser release --clean` を走らせる。ジョブに
  `permissions: contents: write` と `GITHUB_TOKEN` を付与。通常 CI とリリース時の
  二重チェックとする（リリースは低頻度のためテスト再実行コストは許容）。

### 実装上の注意（grill 中に検出）

- **`make test` の前に `make build` が必須。** pytest の
  [tests/conftest.py](tests/conftest.py) `_find_perfmonger_bin()` は固定パス
  `lib/exec/perfmonger_linux_amd64` を探し、無ければ `FileNotFoundError` を送出する。
  `lib/exec/` は gitignore 対象で、Makefile の `test` ターゲットは `build` に
  依存していない。したがって ci.yml / release.yml では `make build && make vet
  && make test` の順で実行する（または `test` を `build` 依存にする Makefile 修正を
  本チケットで行う）。amd64 runner では `make build` の出力名が
  `perfmonger_linux_amd64` となり conftest の探索パスと一致する。
- **goreleaser のビルドと `make build` は別物。** ゲートの `make test` 用には
  `make build`（`lib/exec/` へ出力）が必要で、配布物生成は goreleaser が
  `dist/` に別途行う。両者は共存する（finding #2 の `dir: core` 指定に注意）。
