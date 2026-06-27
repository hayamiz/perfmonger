---
title: Clean up pre-single-binary build artifacts and stale references
type: chore
priority: medium
status: resolved
created: 2026-06-27
updated: 2026-06-27
---

## Description

PerfMonger is now a single-binary Go implementation: the only build product
that matters is `lib/exec/perfmonger_linux_amd64` (see `CLAUDE.md` and the
root `Makefile`, `BIN_DIR := lib/exec`). However, build artifacts from the
earlier multi-binary / "Phase 1" architecture are still lying around in the
working tree, and several git-tracked files still describe or special-case
that old layout. This ticket is to remove the stale artifacts and reconcile
the tracked references.

### 1. Stray build artifacts in the working tree

`lib/exec/` is git-ignored (`.gitignore:5` = `/lib/exec/`), so these are not
committed, but they pollute the dev environment and are confusing:

- `lib/exec/perfmonger-core_linux_amd64` — the old "Phase 1" core binary.
- `lib/exec/perfmonger-player_linux_amd64` — symlink → `perfmonger-core_linux_amd64`
- `lib/exec/perfmonger-plot-formatter_linux_amd64` — symlink → `perfmonger-core_linux_amd64`
- `lib/exec/perfmonger-recorder_linux_amd64` — symlink → `perfmonger-core_linux_amd64`
- `lib/exec/perfmonger-summarizer_linux_amd64` — symlink → `perfmonger-core_linux_amd64`
- `lib/exec/perfmonger-viewer_linux_amd64` — symlink → `perfmonger-core_linux_amd64`

Only `lib/exec/perfmonger_linux_amd64` (the current single binary) should
remain. The wrapper-symlink + core-binary scheme is dead.

Also outside `lib/exec/`:

- `core/cmd/perfmonger/perfmonger` — a stray build output (currently shows up
  as untracked in `git status`). It is the single binary built in the wrong
  place; confirm it is not needed and remove it. Consider whether the build
  flow / `.gitignore` should guard against it reappearing.

### 2. Stale references in git-tracked files

These should be checked and updated/removed where they describe the
pre-single-binary world:

- **`.gitignore`** — entries that only made sense for the old layout:
  - `/core/perfmonger-player` (line 3)
  - `/core/perfmonger-recorder` (line 4)
  - `/core/perfmonger-summarizer` (line 7)
  - `core/cmd/lib/exec/perfmonger-core_*_*` (line 19)
  - `core/cmd/perfmonger-core/perfmonger-core` (line 20)
  - Plus old Ruby-ext entries (`/ext/perfmonger/perfmonger_record`,
    `/ext/perfmonger/Makefile`) and `core-old/` — confirm whether these
    still correspond to anything that can be produced, or are pure cruft.
- **`AGENTS.md`** — still documents the multi-binary design as current:
  - line 33: "Single-binary entry (Phase 1): `core/cmd/perfmonger-core/perfmonger-core.go`"
  - line 54: "Phase 1: builds `perfmonger-core_<os>_<arch>`; compatibility
    wrappers (`perfmonger-{recorder,player,summarizer,plot-formatter}_<os>_<arch>`)
    invoke the core binary."
  - line 134-135: `CoreFinder` looking up `perfmonger-recorder_linux_amd64`
    under `lib/exec/`, dispatching to `perfmonger-core_<os>_<arch>`, and the
    reference to the removed Ruby `lib/perfmonger/command/core.rb`.
  - line 144: reference to `core/cmd/perfmonger-recorder/perfmonger-recorder.go`.

  These describe a layout that no longer exists and should be rewritten to
  match the single-binary architecture (or the obsolete sections removed).

### Note: what is NOT cruft

The directory/package `core/cmd/perfmonger-core/` (the reusable component
packages: `recorder/`, `player/`, `summarizer/`, `plotformatter/`,
`viewer/`) is **current** — it is the live Go import path used throughout
`core/cmd/perfmonger/*.go` and documented in `CLAUDE.md` /
`doc/architecture.md`. Do not remove or rename it; only the old *binary*
named `perfmonger-core_*` and its wrapper symlinks are obsolete.

### Scope / deliverable

1. Remove the stale binaries and wrapper symlinks from `lib/exec/`, and the
   stray `core/cmd/perfmonger/perfmonger`.
2. Prune `.gitignore` of entries that no longer correspond to any producible
   artifact (after confirming each).
3. Update `AGENTS.md` (and double-check `doc/architecture.md`,
   `RELEASING.md`) so no tracked doc still describes the pre-single-binary
   wrapper/CoreFinder scheme as current.
4. Verify `make build` still produces only `lib/exec/perfmonger_linux_amd64`
   and that the test suite passes.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Claims verified against the repo. Removing stale `lib/exec/` binaries +
  symlinks, the stray `core/cmd/perfmonger/perfmonger`, and pruning the obsolete
  `.gitignore` entries are unambiguous. The authoritative current architecture
  is already captured in `CLAUDE.md` and `doc/architecture.md`, so aligning the
  stale docs to them needs no design choices. One scope correction: `AGENTS.md`
  is far more stale than the ticket first noted — it describes an entire Ruby CLI
  (with `exe/perfmonger`, `lib/perfmonger*.rb`, Rakefile, `bundle`) as current,
  so it needs a full rewrite, not just the 4 cited lines. `doc/architecture.md`
  and `RELEASING.md` are already correct and need no changes.

## Implementation Notes

### 1. Remove stale artifacts from the working tree (DECIDED 2026-06-27: remove all)
All of the following are git-ignored, so their removal is local-environment hygiene
and will NOT appear in any commit — document it as a manual cleanup step in the
resolution.
- Delete `lib/exec/perfmonger-core_linux_amd64`.
- Delete the five wrapper symlinks
  `lib/exec/perfmonger-{player,plot-formatter,recorder,summarizer,viewer}_linux_amd64`.
- Delete the stray `core/cmd/perfmonger/perfmonger`.
- Delete the entire `core-old/` tree (Sep-2025 old Go module: own go.mod/build.sh/
  cmd/subsystem — pre-current-`core/` layout residue).
- Delete the entire `core/cmd/lib/` tree (stray `core/cmd/lib/exec/perfmonger_linux_amd64`
  produced by a misbuild with a wrong relative `-o` path).
- Keep `lib/exec/perfmonger_linux_amd64` (current single binary).
- Keep `spec/data/*` — these 7 files are git-tracked golden fixtures used by the
  pytest suite (`conftest.py`), NOT pre-single-binary cruft.

### 2. Prune `.gitignore` (DECIDED 2026-06-27)
Remove these obsolete entries:
- line 3 `/core/perfmonger-player`
- line 4 `/core/perfmonger-recorder`
- line 7 `/core/perfmonger-summarizer`
- line 19 `core/cmd/lib/exec/perfmonger-core_*_*` (pre-single-binary core-binary name;
  not producible by the current build — REMOVE, not keep as a guard)
- line 20 `core/cmd/perfmonger-core/perfmonger-core` (a binary never built; do NOT
  confuse with the live package dir `core/cmd/perfmonger-core/`)
- line 33 `core/cmd/lib/exec/perfmonger_linux_amd64` — DECIDED to remove. This only
  guards a misbuild path (`go build` run from `core/cmd/` without `cd`ing into
  `core/cmd/perfmonger`). The canonical Makefile/CLAUDE build never produces it, so
  it is treated as vestigial, not a safety net. (The `core/cmd/lib/` tree itself is
  deleted in step 1.)
- Ruby-ext entries `/ext/perfmonger/perfmonger_record` and `/ext/perfmonger/Makefile`
  — `ext/` does not exist (tracked or working tree); drop them.
- `core-old/` — the directory is being deleted (step 1) and is pre-single-binary
  residue; drop its ignore entry too.
Keep all other entries (misc run outputs `*.pdf/*.png/*.gp/*.dat/*.pgr`,
Python/pytest, stray `/go.mod`, `apm_modules/`, `.claude/...`, and the still-valid
`/lib/exec/`).

### 3. Delete `AGENTS.md` (DECIDED 2026-06-27)
`AGENTS.md` documents a Ruby-fronted, multi-binary Phase 1 design that no longer
exists, and it is **not referenced by any tracked file** and is a plain file (not a
symlink). The user has confirmed it is no longer used. **Decision: delete it
outright** rather than rewrite — the authoritative current architecture already
lives in `CLAUDE.md` + `doc/architecture.md`, so a rewritten `AGENTS.md` would only
duplicate them. `git rm AGENTS.md`.

### 4. Harden the Makefile `clean` target (DECIDED 2026-06-27: do it)
`clean` is currently `rm -f $(BIN_DIR)/perfmonger_*`, which only matches the current
`perfmonger_linux_amd64` and leaves the hyphenated `perfmonger-core_*` binary and the
`perfmonger-*` wrapper symlinks behind. Broaden it so a future `make clean` self-heals
the same class of stragglers:

```make
clean:
	rm -f $(BIN_DIR)/perfmonger_* $(BIN_DIR)/perfmonger-* core/cmd/perfmonger/perfmonger
```

Scope note: do NOT add `core-old/` or `core/cmd/lib/` to `clean` — those are one-time
manual deletions (step 1); the current build does not regenerate them, so a recurring
`clean` rule for them would be dead weight.

### 5. Verify
`make clean && make build` leaves only `lib/exec/perfmonger_linux_amd64` under
`lib/exec/`; then run the Go unit tests, `go vet`, and `uv run pytest -v` per
`doc/tickets/CLAUDE.md`.

### Decisions resolved (grill 2026-06-27)
1. `AGENTS.md` → delete, not rewrite (step 3).
2. Physical-deletion scope → remove ALL git-ignored pre-single-binary cruft, incl. the
   newly-found `core-old/` and `core/cmd/lib/` (step 1). Removal is local-only (not in
   any commit) since these paths are git-ignored.
3. `.gitignore` lines 19 & 33 (`core/cmd/lib/exec/...`) → remove, not keep as a
   misbuild guard (step 2).
4. Makefile `clean` → harden to also remove `perfmonger-*` wrappers and the stray
   `core/cmd/perfmonger/perfmonger` (step 4).

No open questions remain.

## Resolution

### Git-tracked file changes (will appear in commit)

- **`AGENTS.md` deleted**: The file documented a Ruby-fronted, multi-binary Phase 1
  architecture that no longer exists. It was not referenced by any tracked file and
  was deleted outright. The authoritative architecture is in `CLAUDE.md` and
  `doc/architecture.md`.

- **`.gitignore` pruned**: Removed 9 obsolete entries:
  - `/ext/perfmonger/perfmonger_record` (Ruby ext; `ext/` directory does not exist)
  - `/ext/perfmonger/Makefile` (Ruby ext; same)
  - `/core/perfmonger-player` (pre-single-binary wrapper)
  - `/core/perfmonger-recorder` (pre-single-binary wrapper)
  - `/core/perfmonger-summarizer` (pre-single-binary wrapper)
  - `core/cmd/lib/exec/perfmonger-core_*_*` (old core binary name)
  - `core/cmd/perfmonger-core/perfmonger-core` (binary never built by current Makefile)
  - `core-old/` (deleted directory; pre-current-`core/` layout residue)
  - `core/cmd/lib/exec/perfmonger_linux_amd64` (misbuild path guard; directory deleted)

- **`Makefile` `clean` target hardened**: Extended from
  `rm -f $(BIN_DIR)/perfmonger_*` to also remove `$(BIN_DIR)/perfmonger-*` and
  `core/cmd/perfmonger/perfmonger`, so future `make clean` self-heals hyphenated
  wrapper artifacts and the stray in-source binary.

### Local-only (git-ignored) artifact removals (no commit)

All of the following were git-ignored and therefore their removal does not appear
in any commit — manual local-environment cleanup only:

- `lib/exec/perfmonger-core_linux_amd64` (old Phase 1 core binary)
- `lib/exec/perfmonger-player_linux_amd64` (symlink → core binary)
- `lib/exec/perfmonger-plot-formatter_linux_amd64` (symlink → core binary)
- `lib/exec/perfmonger-recorder_linux_amd64` (symlink → core binary)
- `lib/exec/perfmonger-summarizer_linux_amd64` (symlink → core binary)
- `lib/exec/perfmonger-viewer_linux_amd64` (symlink → core binary)
- `core/cmd/perfmonger/perfmonger` (stray in-source build output)
- `core-old/` tree (Sep-2025 old Go module; pre-current-`core/` layout residue)
- `core/cmd/lib/` tree (misbuild output from wrong `-o` relative path)

### No new unit tests

This was a build-hygiene / documentation cleanup chore with no new or changed source
code paths, so no new unit tests were written or required.

### Verification results

All 5 verification commands passed:

1. `make clean && make build`: succeeded; `ls lib/exec/` shows only
   `perfmonger_linux_amd64`.
2. `cd core/internal/perfmonger && go test -v -cover`: 21 tests PASS,
   coverage 60.2%.
3. `cd core && go test ./cmd/...`: all packages pass (perfmonger, player,
   plotformatter, recorder, summarizer; viewer has no test files).
4. `go vet`: no issues.
5. `uv run pytest -v`: 41 passed in 26.53s.
