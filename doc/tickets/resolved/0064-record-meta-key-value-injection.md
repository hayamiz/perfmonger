---
title: Add --meta key=value metadata injection to record (.pgr header)
type: feature
priority: medium
status: resolved
created: 2026-08-07
updated: 2026-08-08
---

## Description

Implements caller-supplied metadata for the C-embedding interface
(#0009 D4). Depends on #0062 (RunDirect error return). Lets a C caller
tag a recording with query context (query id, db, SQL, etc.) so
`play`/`summary` can surface it.

### Flag & encoding

- `--meta key=value`, repeatable.
- Both key and value are **percent-encoded (RFC 3986)** so arbitrary
  strings (including `=`, spaces, newlines, non-ASCII) can be carried
  while simple values stay human-readable.
- Parse: split each argument on the **first raw `=`** (a literal `=`
  inside key/value is `%3D`), then percent-decode each side. Reject an
  argument with no `=` as a setup error (exit 1).

### Storage & compatibility

- Add `Meta map[string]string` to `CommonHeader` (decoded values).
- gob field addition is forward/backward compatible: old binaries ignore
  the new field, new binaries read old files with a nil/empty map. **No
  format version bump.**
- `play` (and/or `play --header`) and `summary` display the metadata.

`--meta-fd` for very large values (e.g. full SQL text) is explicitly
deferred to v2 and out of scope here.

### Acceptance

- `--meta a=b --meta 'k=%20has%20spaces%3D'` round-trips through record →
  `.pgr` → `play`/`summary` with correct decoding.
- An old-format `.pgr` (no Meta) still plays; a new `.pgr` still plays on
  a binary built without this change.
- Malformed `--meta` (no `=`) exits 1.
- Covered by pytest (see #0065).

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Fix strategy: worktree (feature spanning recorder header + play/summary).
- Affected files: core/internal/perfmonger/perfmonger.go (CommonHeader),
  core/cmd/perfmonger-core/recorder/recorder.go, core/cmd/perfmonger/record.go,
  play/summary display paths, and tests.
- Notes: Design settled in #0009 (D4), including the RFC 3986
  percent-encoding requirement. #0062 is now landed, so a malformed
  --meta can be reported as a SetupError (exit 1). Add Meta
  map[string]string to CommonHeader
  (perfmonger.go:37); gob field addition is forward/backward compatible so
  no format bump. Wire display into play/summary. Independent of #0063.

## Implementation Notes

Plan and acceptance criteria are in the Description above and #0009 D4.
Split on first raw '=', percent-decode both sides, reject args without
'='. Verify round-trip via pytest (#0065).

## Resolution

Fixed on branch `ticket/0064-record-meta-key-value-injection`
(commit `9548f8c`), not yet landed on `master`.

What changed:
- `CommonHeader` gains `Meta map[string]string` (decoded values) in
  `core/internal/perfmonger/perfmonger.go`; gob field addition, no format
  version bump. Added a `SortedMetaKeys()` helper for deterministic output.
- `RecorderOption.MetaArgs []string`; repeatable `--meta` flag
  (`StringArrayVar`) added in `record.go` and forwarded to the daemon
  child in `--background` re-exec.
- `ParseMetaArgs` splits on the first raw `=` and decodes both sides with
  `url.PathUnescape` (RFC 3986; `+` stays literal). No `=`, or a bad
  percent sequence → `*SetupError` (exit 1), before `os.Create`, so no
  output file is left behind.
- `play` prints `# meta: key = value` to stderr (keeps the stdout JSON
  stream clean); `summary` adds a `meta` object (JSON) / `Metadata:`
  block (text), omitted when empty.

Tests: Go parser round-trip (`%3D`/`%20`/`%2B`/non-ASCII, first-`=`
split, reject no-`=` and bad percent), `CommonHeader.Meta` gob
round-trip + nil compat, and `tests/test_meta.py` (record → play/summary
decode, unobtrusive when absent, malformed → exit 1 with no file).

Evaluator: PASS — Sonnet evaluator (Codex CLI unavailable: 401
Unauthorized). Ran `go test -v -cover`, `go test ./cmd/...`, `go vet`,
`make build`, full pytest 44/44, and exercised the built binary:
decoded values correct (incl. literal `+` vs `%2B`, non-ASCII, `%3D`),
malformed → exit 1 with no file, and bidirectional gob compat with a real
pre-fix binary built from `bf212ed` (old plays new, new plays old/golden
fixtures unchanged). No panics.

human-review: optional (purely additive CLI flag + persisted header
field, forward/backward gob-compatible, well-tested, no
security-sensitive surface) — review skipped as low-risk.

Review-state: ready-to-apply
