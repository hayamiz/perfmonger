---
title: README のインストール手順で curl が not found になる不整合
type: bug
priority: medium
status: open
created: 2026-06-27
updated: 2026-06-28
---

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: 不整合の事実（v1.0.0=tar.gz / `.goreleaser.yaml`=生バイナリ / README=生バイナリ）
  は確定しているが、修正には「生バイナリ配布で確定して新リリースを切る（方針A）」か
  「tar.gz 配布へ戻して README を展開手順に直す（方針B）」かの**配布形式の選択**が必要。
  両者は配布成果物が実質的に異なり、方針A は新リリースの公開（対外的なメンテナ操作）を
  伴う。さらに `RELEASING.md` の smoke-test も生バイナリ前提のため、選んだ方針に
  合わせた追随が要る。よって機械的修正ではなくユーザー判断が必要。

## Description

`README.md` の Installation セクション（"Pre-built binary"）に記載された curl
コマンドを実行すると、ダウンロード対象アセットが存在せず **404 (not found)**
になる。手順どおりにバイナリを入手できない。

### README の現状

`README.md:41` / `README.md:46` は **バージョンなしの生バイナリ**を取得する想定:

```sh
curl -L -o perfmonger https://github.com/hayamiz/perfmonger/releases/download/vX.Y.Z/perfmonger_linux_amd64
curl -L -o perfmonger https://github.com/hayamiz/perfmonger/releases/download/vX.Y.Z/perfmonger_linux_arm64
```

### 実際に公開されている成果物

唯一公開済みのリリース **v1.0.0**（2026-05-29、現状 latest）のアセットは
**バージョン入りの tar.gz アーカイブ**:

- `perfmonger_1.0.0_linux_amd64.tar.gz`
- `perfmonger_1.0.0_linux_arm64.tar.gz`
- `perfmonger-1.0.0.tar.gz`（ソース）
- `checksums.txt`

README の URL（`.../v1.0.0/perfmonger_linux_amd64`）に対応するアセットは存在
しないため 404 になる。

### 不整合の根本原因（三者がズレている）

1. **公開済み v1.0.0** … tar.gz アーカイブ（[0007](resolved/0007-release-pipeline.md)
   の当初設計どおり）。
2. **現在の `.goreleaser.yaml`** … `archives.formats: [binary]` /
   `name_template: "perfmonger_{{ .Os }}_{{ .Arch }}"` に**変更済み**で、
   バージョンなしの生バイナリ `perfmonger_linux_amd64` を publish する設定。
3. **README** … 上記 2 の生バイナリ方式に合わせて書かれている。

つまり「生バイナリ配布」へ goreleaser 設定と README を更新したものの、**その設定
での再リリースがまだ無い**。唯一の公開リリース v1.0.0 は旧 tar.gz のままなので、
README の手順がどの公開リリースに対しても成立しない。

### 修正方針（要判断）

triage で以下のどちらに寄せるか決める:

- **方針A: 生バイナリ配布で確定し、新リリースを切る** —
  現在の `.goreleaser.yaml`（binary format）で `v1.0.1`（または `v1.0.0` 再作成）を
  リリースし、README の生バイナリ URL を実在アセットに一致させる。README は
  ほぼ現状維持でよい（`vX.Y.Z` を実バージョンへ）。
- **方針B: tar.gz 配布へ戻す** —
  `.goreleaser.yaml` を tar.gz（version 入り命名）に戻し、README を
  `curl -L … perfmonger_<ver>_linux_amd64.tar.gz | tar xz` 形式の展開手順に修正。
  公開済み v1.0.0 アセットと一致する。

いずれの場合も併せて確認すべき点:
- README の `vX.Y.Z` プレースホルダの扱い（latest へのリダイレクト URL
  `releases/latest/download/<asset>` を使えばバージョン固定が不要になる）。
- `checksums.txt` による SHA256 検証手順を README に追記するか。
- リリース手順書 `RELEASING.md` と整合させる。

### 関連

- リリースパイプライン: [0007](resolved/0007-release-pipeline.md)
- 自己アップデート機能（同じアセット命名規則に依存）:
  [0059](0059-self-update-from-github-releases.md)

## Implementation Notes

### 不整合の確認（HEAD 時点）

- **v1.0.0 release（GitHub, latest）**: tar.gz のみ
  （`perfmonger_1.0.0_linux_amd64.tar.gz` / `..._arm64.tar.gz` / ソース / `checksums.txt`）。
- **`.goreleaser.yaml`**: `archives.formats: [binary]` +
  `name_template: "perfmonger_{{ .Os }}_{{ .Arch }}"` → 生バイナリ
  `perfmonger_linux_amd64` を publish する設定（同梱ファイルは無し）。
- **`README.md:41,46`**: 生バイナリ `.../vX.Y.Z/perfmonger_linux_amd64` を curl。
- **`RELEASING.md`**: smoke-test が生バイナリ前提（`perfmonger_linux_amd64` を curl）。

→ 生バイナリ方式へ設定・ドキュメントを更新したが、その設定での再リリースが未実施。
唯一の公開リリース v1.0.0 は tar.gz のままで README 手順が成立しない。

### 方針A: 生バイナリ配布で確定（現行 `.goreleaser.yaml` を維持）

変更/作業:
- `README.md` — 生バイナリ例は概ね現状維持。`vX.Y.Z` を実バージョン or
  `releases/latest/download/perfmonger_linux_amd64`（リダイレクト）に置換。
- `RELEASING.md` — 既に生バイナリ前提なら整合済み。プレースホルダ運用を明文化。
- **新リリースを切る**（`v1.0.1` 等）。現行 goreleaser 設定で生バイナリを公開して
  初めて README の URL が実在アセットと一致する（＝対外的なリリース操作が必須）。

長所: 設定・README の変更が最小。`tar xz` 展開が不要でユーザー体験が簡素。
短所: 新リリース公開が必要。COPYING/README/NEWS が成果物に同梱されない
（[0007](resolved/0007-release-pipeline.md) の当初方針 H からの後退）。v1.0.0 の
tar.gz 資産とは命名が非連続。

### 方針B: tar.gz 配布へ戻す（公開済み v1.0.0 と一致）

変更/作業:
- `.goreleaser.yaml` — `formats: [binary]` → `[tar.gz]`、`name_template` を
  バージョン入り（`perfmonger_{{ .Version }}_{{ .Os }}_{{ .Arch }}` 等）に戻し、
  `files:`（`COPYING` / `README.md` / `NEWS.md`）を復活。
- `README.md` — 展開手順へ:
  `curl -L .../vX.Y.Z/perfmonger_X.Y.Z_linux_amd64.tar.gz | tar xz`。
- `RELEASING.md` — smoke-test を tar.gz 展開ベースに修正。

長所: 公開済み v1.0.0 アセットと命名・形式が一致し、新リリース無しでも README が
正しくなる（v1.0.0 を指せる）。[0007](resolved/0007-release-pipeline.md) の同梱ファイル
方針 H に整合。
短所: README/`.goreleaser.yaml`/`RELEASING.md` の変更量が方針A より大きい。
ユーザーに `tar xz` 手順が増える。

### 両方針に共通の小決定

- **バージョンプレースホルダ**: `vX.Y.Z` 手動置換のままにするか、
  `releases/latest/download/<asset>` のリダイレクト URL を例示してバージョン固定を
  不要にするか。
- **SHA256 検証**: `checksums.txt` を使った `sha256sum -c` 手順を README に追記するか
  （基本手順に含める / optional 扱い）。
- **スコープ**: `RELEASING.md` の追随を本チケットに含めるか、別チケットへ分離するか。
- **[0059](0059-self-update-from-github-releases.md) との整合**: 自己アップデートは
  同じアセット命名規則に依存するため、ここで確定する命名を 0059 でも前提にする。

### 決定事項（grill 2026-06-27）

- **D1: 配布形式 = 方針B（tar.gz）で確定。** `.goreleaser.yaml` の `archives` を
  tar.gz 構成へ戻す（[0007](resolved/0007-release-pipeline.md) Resolution と同様。
  ただし命名は D2 を参照）。理由: #0007 方針 H（COPYING/README/NEWS 同梱）を満たし、
  生バイナリ方式ではライセンス文書が成果物から落ちるため。

- **D2: アーカイブ名はバージョン非依存（B-ii）。**
  `name_template: "perfmonger_{{ .Os }}_{{ .Arch }}"`（`.tar.gz` 形式）とし、
  ファイル名にバージョンを含めない。README は `releases/latest/download/<asset>`
  のリダイレクト URL を使い、バージョン番号のメンテを不要にする
  （例: `https://github.com/hayamiz/perfmonger/releases/latest/download/perfmonger_linux_amd64.tar.gz`）。
  注: 公開済み v1.0.0 のアセットはバージョン入り名（`perfmonger_1.0.0_linux_amd64.tar.gz`）
  なので、新名は次リリース（D3）以降で初めて存在する。

- **D3: master から `v1.0.1` を切る。** v1.0.0 以降 master に約20件のバグ修正が
  未リリースで積まれている（#0015〜#0056 系）。これらを正式に届けつつ、新
  `.goreleaser.yaml`（D1/D2）で goreleaser を回せば version-less 名のアセットと
  latest リダイレクトが自然に機能する。`NEWS.md` に 1.0.1 エントリを追加する。
  代替案（v1.0.0 のアセットだけ差し替え）は latest が旧コードのまま残るため不採用。

### 確定後の変更対象（暫定）

- `.goreleaser.yaml` — `archives`: `formats: [tar.gz]`、
  `name_template: "perfmonger_{{ .Os }}_{{ .Arch }}"`、`files`（COPYING/README.md/NEWS.md）復活。
- `README.md` — Installation を tar.gz 展開＋latest リダイレクト URL に修正。
- `NEWS.md` — 1.0.1 エントリ追加（未リリース修正群の要約）。
- `RELEASING.md` — smoke-test を tar.gz 展開ベースに修正（スコープは D5 で確定）。
- リリース操作 — `v1.0.1` タグを push して release.yml を発火（コード変更外の運用作業）。

- **D4: SHA256 検証手順は README に含めない。** 手順を最小に保つ方針。
  `checksums.txt` 自体はリリースに同梱されるので、必要なユーザーは個別に検証可能。

### ユーザーへの確認事項（残）

- **D5**: `RELEASING.md` の修正を本チケットに含めるか、別チケットに分離するか。
- **D6**: [0059](0059-self-update-from-github-releases.md)（自己アップデート）は同じ
  アセット命名（version-less `perfmonger_{os}_{arch}.tar.gz`）を前提にしてよいか。
