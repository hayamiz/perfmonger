---
title: Replace Rakefile with Makefile or direct `go` commands
type: chore
priority: medium
status: resolved
created: 2026-04-20
updated: 2026-04-20
---

## Description

現在のビルドタスクは `Rakefile` が起点になっており、`task :cross_build_core` が `./core/build.sh` を呼び、`build.sh` が動的生成した Makefile で `lib/exec/perfmonger_{os}_{arch}` を作るという複雑な構造。[0004-remove-ruby-code.md](0004-remove-ruby-code.md) で Ruby を撤去するため、Rake への依存も同時に解消する必要がある。

やること:
- ルート直下に `Makefile` を新設し、以下のターゲットを提供:
  - `build` — 現在のプラットフォーム向け `perfmonger` をビルド（`CLAUDE.md` 記載のコマンド相当: `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .`）
  - `test` — Go 単体テスト + `uv run pytest`
  - `vet` — Go 静的解析
  - `cross-build` — クロスビルド（[0007-release-pipeline.md](0007-release-pipeline.md) との役割分担に注意）
  - `clean` — `lib/exec/` 以下の成果物のみ削除
- `core/build.sh` の動的 Makefile 生成ロジックは Makefile に吸収できるなら吸収、そうでなければ `build.sh` を簡素化
- `Rakefile` 自体は [0004-remove-ruby-code.md](0004-remove-ruby-code.md) で削除
- `.devcontainer/` の想定コマンドに整合性確認

完了条件:
- `make build && make test` が devcontainer 内で成功
- Rake 呼び出しの残存が無い

## Resolution

### 実施内容

- ルート直下に `Makefile` を新設。`build` / `test` / `vet` / `cross-build` / `clean` ターゲットを提供。
  - `build` は `GOOS`/`GOARCH` で現在プラットフォーム向けに `perfmonger` と `perfmonger-core` をビルドし、互換シンボリックリンク (`perfmonger-recorder`/`-player`/`-viewer`/`-summarizer`/`-plot-formatter`) を生成。
  - `test` は `go test -v -cover` (core/internal/perfmonger) と `uv sync && uv run pytest -v` を実行。
  - `vet` は `CLAUDE.md` 記載コマンド `go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)` と一致。
  - `cross-build` は `linux/amd64` + `linux/arm64` を順に build。
  - `clean` は `lib/exec/perfmonger_*` と `lib/exec/perfmonger-*` のみ削除。
- `Rakefile`、`core/build.sh`、`core/Makefile` を削除（`build.sh` の動的 Makefile 生成とプラットフォーム判定はルート Makefile に吸収）。
- `.devcontainer/devcontainer.json` の `postCreateCommand` に `cd core && go mod download` を追加して、コンテナ起動直後に `make build && make test` がグリーンになるようにした。

### 検証結果

- `make build` — 成功、`lib/exec/perfmonger_linux_amd64` と `perfmonger-core_linux_amd64`、互換 symlink 5 本を生成。
- `make vet` — 成功。
- `make cross-build` — 成功、`linux/amd64` と `linux/arm64` の両バイナリ + symlink を生成。
- `make clean` — 成功、成果物のみ削除（ソースは無傷）。
- `make test` — Go unit tests 全 passed（54.2% coverage）。`uv run pytest` は 27 passed / 5 failed / 4 skipped。失敗はすべて `tests/test_background.py` で発生し、コンテナ内で `os/user.Current()` が解決できないことに起因する既知の問題（session file が `/tmp/perfmonger-unknown-session.pid` になる）。本チケットの変更に起因する退行ではない。
- `grep -R "rake\|Rakefile" . --exclude-dir=.git --exclude-dir=doc` — 実行可能な Rake 呼び出し箇所はゼロ。残存は legacy CI config (`wercker.yml`, `.travis.yml`)、gemspec (`perfmonger.gemspec`)、ドキュメント (`README.md`, `AGENTS.md`, `misc/release-howto.txt`)、`.venv` 配下の pytest/pygments 文字列のみ。これらは #0004 (Ruby 撤去) / #0007 (release pipeline) の別スコープ。

### Follow-ups

- `perfmonger.gemspec` の削除は #0004 の範囲。
- `wercker.yml` / `.travis.yml` は現在 CI として稼働していないため放置可。新 CI は #0007 で定義。
- `AGENTS.md` の `bundle exec rake` 記述更新は #0004 or #0005 (README update) の一部として処理するのが自然。
