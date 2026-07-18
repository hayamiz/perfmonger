---
title: Go ツールチェーンを 1.25.12+ に引き上げて残存 stdlib 脆弱性を解消
type: chore
priority: high
status: resolved
created: 2026-07-12
updated: 2026-07-12
---

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: 修正対象・修正値ともに govulncheck 調査で確定済み（下記 Description 参照）。
  配布形式や設計判断は不要で、`go` ディレクティブ／CI の `go-version`／ドキュメントの
  バージョン記述を機械的に引き上げるだけ。依存モジュール（x/crypto・x/sys）は
  Dependabot PR #44 で解決済みのため追加作業なし。検証はプロジェクト規定の
  `make build && make vet && make test` と govulncheck で行う。

## Description

`govulncheck` による調査の結果、`core` モジュールで**到達可能な脆弱性はすべて
Go 標準ライブラリ由来**であり、ビルドに使う Go ツールチェーンを引き上げることで
解消できることが判明した。ローカル開発環境（go1.26.5）ではすでに到達可能な
脆弱性は 0 件だが、**CI・リリースビルドは古い Go を使うため配布バイナリには
脆弱性が残る**。この恒久対処を行う。

### 現状のギャップ

- `core/go.mod`: `go 1.25.0`
- `.github/workflows/ci.yml` / `release.yml`: `go-version: '1.24'`
- `RELEASING.md`: 「Go 1.24」、`README.md`: 「Go 1.24 or later」

`GOTOOLCHAIN=auto` の下、CI は `go` ディレクティブの `1.25.0` までしか自動昇格
しない。`1.25.0` は下記の修正を含まないため、CI 産の配布バイナリに到達可能な
脆弱性（特に `GO-2026-4602`）が残る。

### govulncheck が示した到達可能な脆弱性（go1.24 ビルド時点）

| ID | パッケージ | 修正版 | 備考 |
|----|-----------|--------|------|
| GO-2026-4602 | `os`（FileInfo が Root から脱出） | os@go1.25.8 | 最も高い下限の一つ |
| GO-2025-3956 | `os/exec`（LookPath） | os/exec@go1.24.6 | go1.25 系で解消済み |
| GO-2025-3750 | `os`/`syscall`（O_CREATE\|O_EXCL） | os@go1.24.4 | Windows 限定・Linux 専用の本 project では無影響 |

到達不能層（import/require のみ）にも stdlib のパッチ待ち advisory が多数あり、
その修正下限の最大は `1.25.12`（例: `os@go1.25.12`、`stdlib@go1.25.12`）。

### 決定した修正値

ビルド下限を **`1.25.12`** に引き上げる（`GO-2026-4602` の 1.25.8 を含み、
到達不能な stdlib advisory 群もまとめて解消できる最小の統一値）。

### 修正内容

1. `core/go.mod` — `go 1.25.0` → `go 1.25.12`（`GOTOOLCHAIN=auto` の下限を固定）。
2. `.github/workflows/ci.yml` / `.github/workflows/release.yml` —
   `go-version: '1.24'` → `'1.25'`（既存流儀どおり最新 1.25.x に追随）。
3. `RELEASING.md` — 「Sets up Go 1.24」等の記述を Go 1.25 系へ更新。
4. `README.md` — Build from source の「Go 1.24 or later」を「Go 1.25 or later」へ。

### 残る非対処項目（対処不能・対処不要）

- `GO-2026-5932`（`golang.org/x/crypto@v0.52.0`、Fixed in: N/A）— 上流に修正版が
  存在せず、かつ perfmonger の実コードから到達不能。バージョン更新では消せない
  ため本チケットのスコープ外。

### 関連

- 依存更新（x/crypto 0.45.0→0.52.0）: Dependabot PR #44（マージ済み）
- ローカル goreleaser ツーリング: コミット `34aafb7`

## Resolution (2026-07-12)

Description の「修正内容」1〜4 を機械的に適用した。

### 変更ファイル

- `core/go.mod` — `go 1.25.0` → `go 1.25.12`。`GOTOOLCHAIN=auto` のビルド下限を
  固定し、CI・リリースでも到達可能だった `GO-2026-4602`（要 1.25.8）を含む
  stdlib 脆弱性群を解消する。
- `.github/workflows/ci.yml` / `.github/workflows/release.yml` —
  `go-version: '1.24'` → `'1.25'`。
- `RELEASING.md` — 「Sets up Go 1.24」→「Sets up Go 1.25」。
- `README.md` — Build from source の「Go 1.24 or later」→「Go 1.25 or later」。

### 検証（go1.26.5 ローカル環境、go.mod 下限 1.25.12）

- `make build` … PASS（`make clean` 後のフルビルド）。
- `make vet` … PASS。
- `make test`（Go 単体 + pytest）… PASS（pytest 41 passed）。
- `govulncheck ./...`（core）… **到達可能な脆弱性 0 件**。到達不能な残 1 件は
  `GO-2026-5932`（`golang.org/x/crypto@v0.52.0`、Fixed in: N/A）で、上流に修正が
  無く実コードから未到達のため対処不能・対処不要（Description 参照）。

### 備考

コード変更は無く、ビルドツールチェーン下限とドキュメントのバージョン記述のみの
更新。ユーザー可視の動作変更は無いため spec 更新は不要（本 project に spec 宣言も
なし）。
