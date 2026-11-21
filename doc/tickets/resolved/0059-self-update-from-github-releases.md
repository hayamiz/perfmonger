---
title: GitHub Releases を使った自己アップデート機能 (perfmonger --update)
type: feature
priority: medium
status: resolved
created: 2026-06-27
updated: 2026-11-21
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

## Triage

- Complexity: medium
- Mechanical fix: yes（D1/D2 の決定後は Implementation Notes の計画どおりに実装でき、残りは推奨デフォルトで確定）
- Requires user decision: no（D1/D2 は 2026-11-21 にユーザーが決定済み）
- Affected files: 約 8（新規: `core/cmd/perfmonger/systemupdate.go`, `core/cmd/perfmonger/systemupdate_test.go`, `tests/test_system_update.py`／変更: `core/cmd/perfmonger/main.go`, `core/cmd/perfmonger/initshell.go`, `README.md`, `doc/architecture.md`, `CLAUDE.md`）
- Fix strategy: worktree
- Notes: 方式はユーザーが確定済み（catscope の `updater.go` に倣った自前実装、GitHub Releases の成果物取得、自己置換、`checksums.txt` の SHA256 検証）。新規サブコマンドとして閉じた変更で既存機能への影響は小さいが、ネットワークとバイナリ置換を扱うためテスト込みで 300〜400 行規模。コマンド名（D1: `system-update`）と `dev` ビルドの扱い（D2: 拒否）はユーザーが決定済みで、他は下記の推奨デフォルトで進める。

## Implementation Notes

### 前提（ユーザー確定事項, 2026-11-21）

- https://github.com/hayamiz/catscope の `updater.go` 方式を踏襲した自前実装（サードパーティの self-update ライブラリは使わない）。
- GitHub Releases から成果物を取得し、実行中のバイナリを同じ場所で置き換える。
- ダウンロード物はリリースの `checksums.txt`（SHA256）で必ず検証する。

### catscope 実装との差分

catscope は `--system-update` フラグで `releases/latest/download/catscope-linux-amd64`（生バイナリ、amd64 固定）を `http.Get`（タイムアウトなし）で取得し、同じディレクトリの temp file → 元パーミッションへ chmod → `os.Rename` で置き換え、rename 後に `--version` を実行する。checksum 検証・バージョン比較・確認プロンプトはない。本実装ではこれらを補う。

### 処理フロー（推奨案）

1. 環境判定: `runtime.GOOS == "linux"` かつ `runtime.GOARCH` が `amd64`/`arm64` なら asset 名 `perfmonger_linux_<arch>.tar.gz`。それ以外はエラー。
2. バージョン判定: `VERSION` が `dev` または `X.Y.Z` として解析できなければ、ネットワークアクセス前に拒否（D2）。
3. 最新タグ解決: リダイレクトを追わない client で `https://github.com/hayamiz/perfmonger/releases/latest` を取得し、`Location` の `.../releases/tag/vX.Y.Z` からタグを得る。以降は `.../releases/download/<tag>/<asset>` にタグを固定して取得する（`latest/download` を 2 回叩くと公開途中で checksums と tarball の版がずれ得るため）。
4. 比較: latest ≤ current なら `up to date` を表示して exit 0（ダウンロードしない）。`--check` は current/latest を表示して exit 0。
5. 書き込み可否の事前確認: `os.Executable()` → `filepath.EvalSymlinks` で実体を求め、同じディレクトリに `os.CreateTemp(dir, ".perfmonger-update-*")` を作れるか確認。不可なら sudo を案内してプロンプト前に失敗させる。
6. ダウンロードと検証: `checksums.txt`（上限 1 MiB）と tarball（上限 64 MiB）を取得し、tarball の SHA256 を `checksums.txt` の該当行と照合。該当行なし・不一致なら中断。検証は展開より前。
7. 展開: tar.gz から通常ファイル `perfmonger`（`./perfmonger` も許容）を temp file へ（上限 128 MiB）。同梱の `NEWS.md` をメモリに読む。
8. リリースノート表示: `NEWS.md` を `## YYYY-MM-DD: PerfMonger X.Y.Z` の見出しで節に分け、`current < v <= latest` の節を表示。読めなければ警告して続行。
9. 確認: `Update perfmonger X -> Y? [y/N]`（既定 N）。`--yes`/`-y` で省略。stdin が TTY でなく `--yes` もなければエラー。
10. 置換: temp を fsync → close → 元パーミッションへ chmod → rename 前に `<tmp> --version` で latest を含むことを確認 → `os.Rename`。失敗時は defer で temp を削除。
11. 横断: SIGINT/SIGTERM で ctx をキャンセル（temp 削除が走る）、`http.Client` にタイムアウト、User-Agent `perfmonger/<VERSION>`、200 以外はエラー。

### テスト容易性

updater を struct にまとめ、`baseURL`・`httpClient`・`exePath`・`currentVersion`・`goos`/`goarch`・stdin/stdout・TTY 判定を注入可能にする。Go unit test は `httptest.NewServer` がテスト内で組み立てた tarball と checksums を返す（`/latest` は 302）。バージョン比較、checksums パース・不一致拒否、tar 抽出（エントリなし・上限超過）、NEWS 節抽出、パーミッション保持での置換、書き込み不可、未対応 arch、非 TTY で `--yes` なし、up to date／`--check` で置換しないこと、スモークテスト失敗で置換しないこと、`expectedFlags` を確認する。pytest はネットワーク不要な範囲（`--help`、`dev` ビルドの拒否）に限る。

### 判断ポイントと推奨デフォルト

- **D1【決定】CLI 形態**: `perfmonger system-update` サブコマンド（ユーザー決定, 2026-11-21）。catscope の `--system-update` と同じ名前を、既存の cobra サブコマンド構成（1 ファイル 1 コマンド・help・補完）に載せる。フラグは `--check`（確認のみ）と `--yes`/`-y`（プロンプト省略）。
- **D2【決定】`dev` ビルド**: 拒否する（exit 1、手動インストールを案内）（ユーザー決定, 2026-11-21）。`make build` の `lib/exec/perfmonger_linux_amd64` がリリース版に置き換わり pytest がリリース版を検証する事故を防ぐ。
- **D3 バージョン比較**: `X.Y.Z` の数値比較を自前実装（依存追加なし）。`-rc1` 等の suffix は解析不能としてエラー。downgrade はしない。
- **D4 最新版とノートの取得経路**: GitHub API は使わず、`/releases/latest` のリダイレクトでタグを解決し、検証済み tarball 同梱の `NEWS.md` からノートを抽出する（未認証 API のレート制限・JSON 処理が不要）。なお v1.0.0/v1.1.0 の GitHub Release 本文は `.goreleaser.yaml` の `changelog: disable: true` の影響で空になっており（別途対応）、API の `body` に頼らない点でもこの方式が堅い。
- **D5 確認プロンプト**: ノート表示後に `[y/N]`、`--yes`/`-y` で省略、非 TTY で `--yes` なしはエラー。
- **D6 追加フラグ**: `--check` と `--yes` のみ。バージョン指定の pin/downgrade と `--force` はスコープ外。
- **D7 未対応環境**: linux 以外・amd64/arm64 以外はエラー（`runtime.GOARCH` は GoReleaser の `{{ .Arch }}` と一致）。
- **D8 検証**: `checksums.txt` の SHA256 照合まで。GPG/cosign は対象外。
- **D9 置換方式**: 上記フロー 10。symlink は実体を置き換え、所有者は保持しない（catscope と同じ）。
- **D10 pager**: 使わない（直後に対話プロンプトがあり、#0001 の実装は `summaryCommand` に密結合）。
- **D11 安全策**: サイズ上限、HTTP タイムアウト、SIGINT での temp 削除、User-Agent。

### 補足

v1.1.0 以前のバイナリには本機能がなく、v1.0.0 は asset 名にバージョンを含むため自己更新できない。自己更新が実際に使えるのは、本機能を含む最初のリリースから次のリリースへの更新以降。手動確認は `-ldflags "-X main.VERSION=1.0.0"` でビルドしたバイナリを一時ディレクトリに置き、実際の v1.1.0 へ更新されることで行える。

## Resolution

ブランチ `ticket/0059-self-update-from-github-releases`（commit `ca327c1`、master `67d082e` = v1.1.0 から分岐）に実装。まだ master には入っていない。

### 変更内容

- `core/cmd/perfmonger/systemupdate.go`（新規）: `perfmonger system-update` サブコマンド（`--check`、`--yes`/`-y`）と updater 本体。
  - `systemUpdater` struct に `BaseURL`・`HTTPClient`・`ExePath`・`CurrentVersion`・`GOOS`/`GOARCH`・stdin/stdout/stderr・TTY 判定・サイズ上限を注入でき、本番値は `newSystemUpdateCommand()` が設定する。
  - 未対応 os/arch、`dev`、`X.Y.Z` でないバージョンはネットワークアクセス前に exit 1（D2）。
  - `/releases/latest` のリダイレクト（追わない）からタグを取り出して `^v\d+\.\d+\.\d+$` で検査し、`checksums.txt` と `perfmonger_linux_<arch>.tar.gz` をそのタグに固定して取得する。
  - 順序: バージョン比較（`--check`／up to date はここで exit 0、ダウンロードなし）→ 実体パスの書き込み可否確認（不可なら sudo を案内）→ 非 TTY かつ `--yes` なしはエラー → ダウンロード → 展開前に SHA256 照合 → 通常ファイル `perfmonger` だけを同じディレクトリの temp に展開、`NEWS.md` はメモリに読む → `current < v <= latest` の NEWS 節を表示 → `[y/N]`（既定 N）→ fsync・chmod（元のパーミッション）→ `<tmp> --version` が latest を報告することを確認 → `os.Rename` → ディレクトリ fsync。rename 前のどの失敗でも temp を削除する。
  - HTTP: 全体 5 分・ヘッダ 30 秒・TLS 15 秒のタイムアウト、User-Agent `perfmonger/<VERSION>`、200 以外はエラー、https→http のリダイレクトは拒否。`signal.NotifyContext(SIGINT, SIGTERM)` で中断し、プロンプト中の Ctrl-C でも元のバイナリは変わらない。
  - アーカイブ: `perfmonger` 以外のエントリ（`../perfmonger`、ネストしたパス、絶対パス等）はディスクに書かない。`perfmonger` が symlink/hardlink/dir の場合、重複、上限超過はエラー。
- `core/cmd/perfmonger/main.go`: サブコマンドを登録。
- `core/cmd/perfmonger/initshell.go`: bash/zsh 補完に `system-update`（zsh は `--check`/`-y`/`--yes` も）を追加。
- `README.md`（Installation に「Updating」節）、`doc/architecture.md`（ファイル構成・サブコマンド表・§5.9・テスト一覧）、`CLAUDE.md`（サブコマンド一覧）を更新。
- 推奨案からの差分: 非 TTY チェックをダウンロード前に移動（拒否が確定しているのに取得しないため）。`/releases/latest` は 301/302/303/307/308 を受け付ける。`dev` は `--check` も拒否。`checksums.txt` は不正な行が 1 つでもあればエラー。

### テスト

- `core/cmd/perfmonger/systemupdate_test.go`（新規、31 関数）: バージョン解析と比較、タグ検証、checksums の解析（欠落・不一致・不正行・重複）、アーカイブ抽出（エントリなし、symlink/hardlink/dir、`../perfmonger` などのパス、上限超過、重複）、NEWS 節抽出、`resolveLatestTag` の各ケース、httptest による更新（パーミッション保持、symlink 先の置換、up to date／`--check` はダウンロードしない）、16 の失敗ケースで元のバイナリが不変かつ temp が残らないこと、書き込み不可・非 TTY・プロンプト回答・キャンセルの各経路、`expectedFlags`。
- `core/cmd/perfmonger/initshell_test.go`: 登録済みの全サブコマンドが bash/zsh の補完に載っていることを確認するテストを追加。
- `tests/test_system_update.py`（新規、ネットワーク不要）: `--help`、`dev` ビルドの拒否（exit 1・バイナリ不変）など 5 件。

### 検証

- `make -B build`、`go test -v -cover`（internal、67.4%）、`go test ./cmd/...`（全 ok）、`go vet`（2 種）、`uv run pytest -v`（64 passed、1 skipped）がすべて通過。
- 実際の v1.1.0 リリースに対する E2E: `-X main.VERSION=1.0.0` でビルドしたバイナリで `--check`（exit 0、ダウンロードなし）、`</dev/null` で `--yes` なし（exit 1）、`--yes` で 1.1.0 へ更新（パーミッション 0751 を保持、temp なし、置換後の sha256 が公式 tarball 内の `perfmonger` と一致）、疑似 TTY で `n`（不変）・Ctrl-C（不変・temp なし）を確認。

Evaluator: PASS — Sonnet evaluator（ticket:ticket-evaluator）。Codex CLI は 401 Unauthorized（未認証）で結果が得られず fallback。evaluator は検証コマンド 6 種をすべて実行して通過を確認し、実リリースでの E2E（`--check`、非 TTY 拒否、疑似 TTY で `n`、symlink 経由の `-y` 更新、`dev` 拒否）と、3 種の mutation（checksum 比較の無効化・通常ファイル検査の無効化・スモークテスト削除）がそれぞれテストで検出されることを確認した。非ブロッキングの指摘: プロンプト中断時に stdin 読み取りの goroutine が残る（直後にプロセスが終了するので無害）、非 TTY チェック前に temp を作る（defer で削除される）、checksums と tarball が同じ Release から来るため Release 自体の改ざんは防げない（要件上の限界、署名は 0007 の follow-up）。
human-review: recommended
Human review: approved（2026-11-21、crit で `67d082e..ticket/0059-self-update-from-github-releases` の 9 ファイルを確認、コメントなし）
Review-state: ready-to-apply

### フォローアップ

- 自己更新が実際に使えるのは本機能を含む最初のリリースからその次のリリースへの更新以降（v1.1.0 以前には本機能がない）。
- arm64 は asset 名の対応をユニットテストで確認したのみで、実機 E2E は未実施。
- `make test` は `core/cmd/...` のテストを実行しない（既存の制約、今回は変更なし）。
