---
title: Replace Rakefile with Makefile or direct `go` commands
type: chore
priority: medium
status: open
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
