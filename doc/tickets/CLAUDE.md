# Ticket conventions

This directory holds file-based work-item tickets. Each ticket is a Markdown
file with YAML frontmatter. Tickets are managed by the `ticket` plugin
(`/ticket-create`, `/ticket-check`, `/ticket-triage`, `/ticket-fix`).

## File naming

- Open / in-progress / blocked tickets: `doc/tickets/NNNN-<kebab-subject>.md`
- Resolved tickets: `doc/tickets/resolved/NNNN-<kebab-subject>.md`
- `NNNN` is a zero-padded 4-digit sequence; never reuse numbers.
- `<kebab-subject>` is 2–5 words in kebab-case.

## Frontmatter schema

```yaml
---
title: <one-line human-readable title>
type: bug | feature | enhancement | refactor | docs | test | chore
priority: critical | high | medium | low
status: open | in-progress | blocked | resolved
created: YYYY-MM-DD
updated: YYYY-MM-DD
---
```

## Body sections

Required:

- `## Description` — what and why. Include enough context that a reader
  unfamiliar with the conversation can act on it.

Added by `/ticket-triage`:

- `## Triage`
  - `Complexity: low | medium | high`
  - `Mechanical fix: yes | no`
  - `Requires user decision: yes | no`
  - `Notes:` a short rationale.

Added by `/ticket-triage` when `Mechanical fix: no`:

- `## Implementation Notes` — concrete plan, alternatives, open questions,
  specific decision points for the user.

Added by `/ticket-fix` on resolution:

- `## Resolution` — what was changed, which tests were added, any follow-ups.

## Lifecycle

- `open` — newly created, not yet triaged.
- `in-progress` — being worked on (set by `/ticket-fix`).
- `blocked` — waiting on external input; keep in the open directory.
- `resolved` — done; file moves to `doc/tickets/resolved/`.

## Project integration

If this project has a spec or design doc that tickets should stay consistent
with, name it here (e.g., `Spec: doc/SPEC.md`). `/ticket-fix` will read this
hint and update the spec when a fix changes user-visible behavior. If no
spec is declared, the spec-update step is skipped.

## Verification

`/ticket-fix` should run the following after implementing a fix:

```bash
# Go unit tests
cd core/internal/perfmonger && go test -v -cover

# Go static analysis
cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)

# Integration tests
uv sync && uv run pytest -v
```
