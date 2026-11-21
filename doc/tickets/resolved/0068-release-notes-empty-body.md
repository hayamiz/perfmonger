---
title: GitHub Release の本文が空になる（changelog.disable が --release-notes ごとスキップ）
type: bug
priority: high
status: resolved
created: 2026-11-21
updated: 2026-11-21
---

## Description

v1.0.0 / v1.1.0 の GitHub Release の本文が空になっている（Releases API の
`body` がそれぞれ `"\n"` / `""`）。`release.yml` は `NEWS.md` の先頭節を
`/tmp/relnotes.md` に抜き出して `goreleaser release --release-notes=/tmp/relnotes.md`
に渡しているが、その内容が Release に入っていない。

### 原因

`.goreleaser.yaml` の次の設定:

```yaml
changelog:
  disable: true
```

goreleaser v2（v2.17.0 で確認）の `internal/pipe/changelog/changelog.go` では、

- `Skip()` が `ctx.Config.Changelog.Disable` を返すため、`disable: true` だと
  changelog パイプ全体がスキップされる。
- `--release-notes` で渡したファイルの読み込み（`loadContent(ctx, ctx.ReleaseNotesFile, ...)`
  → `ctx.ReleaseNotes`）は、そのパイプの `Run()` の冒頭で行われている。

つまり git log ベースの changelog を止めるつもりの設定が、`--release-notes` の
読み込みまで止めてしまっている。`--release-notes` を指定した場合は `Run()` が
ノートを読み込んだ直後に早期 return するので、`disable` を外しても git log の
changelog は生成されない。

### 確認済みの事実

ローカルで v1.1.0 タグ上の HEAD に対し
`goreleaser release --clean --skip=publish,announce --release-notes=<NEWS 先頭節>`
を実行し、設定の有無でパイプラインを比較した。

- `disable: true` あり（現状）: `generating changelog` の段が実行されない。
- `disable` 行なし: `generating changelog` の段が実行される。

### 修正内容

1. `.goreleaser.yaml` から `changelog:` / `disable: true` を削除し、直前の
   コメント（"disable goreleaser's git-log based changelog (D3)"）を、
   `--release-notes` 指定時は git log の changelog が生成されない旨と、
   `disable: true` にすると `--release-notes` も読まれなくなる旨に書き換える。
2. `RELEASING.md` の「GoReleaser's own git-log changelog is disabled」の記述を
   実態に合わせて直す（必要なら）。

### 公開済み Release の扱い

公開済み v1.1.0 の本文は、ユーザーがブラウザで Release を編集し、`NEWS.md`
の 1.1.0 節を貼り付けて直す（ユーザー決定, 2026-11-21）。v1.0.0 も同様に
直せるが任意。

### 関連

- リリースパイプライン: [0007](resolved/0007-release-pipeline.md)（D3: NEWS.md を
  リリースノートの source にする決定）
- 自己更新: [0059](0059-self-update-from-github-releases.md)（ノートは Release 本文ではなく
  tarball 同梱の `NEWS.md` から読む方針）

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Affected files: 1〜2（`.goreleaser.yaml`、必要なら `RELEASING.md`）
- Fix strategy: in-place
- Notes: 原因と修正をソースと実機の dry-run で確認済み。設定 2 行の削除とコメント修正のみ。本番での最終確認は次回リリースの Release 本文で行う。

## Resolution

### 変更内容

- `.goreleaser.yaml`: `changelog:` / `disable: true` を削除し、直前のコメントを
  「`--release-notes` 指定時は git log の changelog は生成されない。`disable: true`
  にすると changelog パイプごと `--release-notes` の読み込みもスキップされ、Release
  本文が空になる（#0068）」という内容に書き換えた。
- `RELEASING.md`: 「GoReleaser's own git-log changelog is disabled」の記述を、
  `--release-notes` でノートを渡すため git log の changelog は生成されない、という
  実態に合わせて修正した。

### 回帰テスト

`tests/test_release_config.py` を追加した（テキスト/正規表現ベース、追加依存なし）。
`release.yml` が `--release-notes` を渡していること、`.goreleaser.yaml` のトップレベル
`changelog:` に `disable: true` がないことを検査する。旧設定
（`git stash` で `.goreleaser.yaml` のみ戻した状態）では
`test_goreleaser_does_not_disable_changelog` が失敗し、新設定では 3 件とも通ることを確認した。

### 検証結果

- `go test -v -cover`（core/internal/perfmonger）: PASS（coverage 67.4%）
- `go test ./cmd/...`（core）: 全 ok
- `go vet`: 問題なし
- `uv run pytest -v`: 62 passed, 1 skipped
- `goreleaser check`: 1 configuration file(s) validated
- dry-run（`--skip=publish,announce,validate --release-notes=<NEWS 先頭節>`）:
  `generating changelog` の段が出力され、`release succeeded after 2s`。

### 後続対応

- 次回の実リリースで、GitHub Release の本文が空でないことを確認すること。
- 公開済み v1.1.0（任意で v1.0.0）の本文は、ユーザーがブラウザで `NEWS.md` の該当節を
  貼り付けて手動で修正する（ユーザー決定済み）。
