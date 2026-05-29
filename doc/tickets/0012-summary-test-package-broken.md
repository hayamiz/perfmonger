---
title: core/cmd/perfmonger test package does not compile (validatePagerOption undefined)
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-05-29
---

## Description

`core/cmd/perfmonger` のテストパッケージが **コンパイルできない**状態になっている。
`go vet ./cmd/perfmonger/`（および `go test ./cmd/perfmonger/`）が以下で失敗する:

```
vet: cmd/perfmonger/summary_test.go:131:22: summaryCmd.validatePagerOption
  undefined (type *summaryCommand has no field or method validatePagerOption)
```

[core/cmd/perfmonger/summary_test.go](core/cmd/perfmonger/summary_test.go) の
`TestSummaryCommand_ValidatePagerOption`（83 行〜）が
`summaryCmd.validatePagerOption(cobraCmd)` を呼び出しているが、
[core/cmd/perfmonger/summary.go](core/cmd/perfmonger/summary.go) に
`validatePagerOption` メソッドは存在しない。現状あるのは
`resolvePager()`（64 行）と `runWithPager()`（93 行）のみ。

おそらく #0001（summary のページャ対応）実装時に、テストが想定するメソッド名・
シグネチャと実装がずれた（リネームされた／実装が漏れた）もの。テストが
コンパイルを通さないため、当該パッケージのユニットテストは **1 つも実行されていない**。

### 影響

- `core/cmd/perfmonger` 配下のテスト（`play_test.go`, `record_test.go`,
  `stat_test.go`, `summary_test.go`, および #0007 で追加された `main_test.go`）が
  すべてビルドできず実行不能。
- プロジェクト規定の検証コマンド（`doc/tickets/CLAUDE.md` の `## Verification`）は
  `core/internal/perfmonger` のみを対象としているため、この破損は CI / triage の
  既定フローでは検出されていなかった。#0007 実装中に発見。

### 発見経緯

#0007（リリースパイプライン）で `core/cmd/perfmonger/main_test.go` を追加した際、
当該パッケージが既存の `validatePagerOption` 未定義でコンパイルできないことが判明。
HEAD（#0007 着手前）の `summary_test.go` でも同じ参照が存在し、`git grep` で
`func validatePagerOption` が定義されていないことを確認済み → #0007 とは無関係の
既存バグ。

## 期待される対応（要調査）

以下のいずれかが妥当。実装時に #0001 の意図を確認すること:

1. **実装側を追加**: `summaryCommand` に `validatePagerOption(cmd *cobra.Command) error`
   を実装する（テストが期待する挙動 = ページャ関連オプションの検証。
   `summary_test.go:83-145` の各ケースが仕様の手がかり）。
2. **テスト側を現行 API に合わせる**: 検証ロジックが既に `resolvePager()` 等に
   統合されているなら、テストを現行メソッドに合わせて書き換える。

どちらにせよ、最終的に `cd core && go test ./cmd/perfmonger/` が通ること。

## 受け入れ条件

- `cd core && go vet ./cmd/perfmonger/` が通る。
- `cd core && go test ./cmd/perfmonger/` がビルド・実行され、緑になる。
- 可能なら `doc/tickets/CLAUDE.md` の `## Verification` に
  `core/cmd/perfmonger` のテストも追加し、再発を防ぐ。
