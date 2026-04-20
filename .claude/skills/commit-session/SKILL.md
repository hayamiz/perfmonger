---
name: commit-session
description: "Commit files changed in the current Claude Code session, grouped into semantically coherent commits."
argument-hint: "[--dry-run]"
allowed-tools: Bash(*) Read Glob Grep
---

# Commit Session

Commit only the files that were changed during the current Claude Code session, organized into semantically coherent groups. Each group becomes a separate commit with a descriptive message.

## Arguments

Parse `$ARGUMENTS` for:
- `--dry-run` (optional): Show the planned commit groups without actually committing. Default: false.

## Execution Flow

### Step 1: Identify Session Changes

Determine which files were changed in this session by examining the conversation context.

Scan the conversation history for:
1. **Files written or edited** — any file paths used with Write or Edit tools in this session.
2. **Files created via Bash** — any files created by shell commands (redirects, `cp`, `mv`, script outputs).

Cross-reference with `git status --porcelain` to confirm which of these files actually have uncommitted changes. Only include files that:
- Appear in `git status` (modified, new, or deleted), AND
- Were touched during this session

Present the list:
```
## Session Changes

Modified:
  - path/to/file1
  - path/to/file2

New:
  - path/to/file3

Deleted:
  - path/to/file4

Not changed by this session (skipped):
  - path/to/other_dirty_file (dirty but not from this session)
```

### Step 2: Semantic Grouping

Group the session-changed files into semantically coherent commits. Criteria:

1. **By functional area**: Files that belong to the same feature, module, or subsystem go together.
   - Example: a script change and its corresponding test → one commit
   - Example: related agent prompt updates → one commit
2. **By change type**: If files span multiple areas but share a change type:
   - Documentation updates
   - Configuration changes
   - Test changes
   - Code/script changes
   - Agent prompt changes
   - Skill definitions
3. **By dependency**: If file A's changes depend on file B's changes, they go in the same commit or in dependency order.

Use the conversation context to understand the purpose of each change and group accordingly. Changes made for the same user request or goal naturally belong together.

Present the grouping plan:

```
## Planned Commits

### Commit 1: <summary>
Files:
  - path/to/file1 (modified)
  - path/to/file2 (new)
Message: "<draft commit message>"

### Commit 2: <summary>
Files:
  - path/to/file3 (modified)
Message: "<draft commit message>"
```

### Step 3: Confirm

If `--dry-run`, stop here.

Otherwise, ask the user to confirm the plan. Accept adjustments (move files between groups, change messages, include/exclude files).

### Step 4: Execute Commits

For each commit group, in order:

1. Stage only the files in that group: `git add <file1> <file2> ...`
2. Commit with the agreed message, appending `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`.
3. Verify with `git status` that the expected files were committed.

Use heredoc format for commit messages:
```bash
git commit -m "$(cat <<'EOF'
<commit message>

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

### Step 5: Summary

Report:
```
## Commit Summary

| # | Commit | Files | Hash |
|---|--------|-------|------|
| 1 | <message> | N files | abc1234 |
| 2 | <message> | N files | def5678 |
...

Total: N commits, M files committed
Remaining dirty files: K (not from this session)
```

## Project-Specific Context

This skill is project-agnostic. Do NOT hardcode project-specific directory names, file patterns, or conventions into this file. Instead:

- Read `CLAUDE.md` at runtime for project directory structure, conventions, and testing policies.
- Read `.gitignore` at runtime for project-specific exclusion patterns.
- Use MEMORY.md and conversation context for any additional project knowledge.

This keeps the skill portable across repositories.

## Notes

- Never use `git add -A` or `git add .` — always add specific files.
- Never amend existing commits — always create new commits.
- If a pre-commit hook fails, diagnose and fix, then create a new commit (do not use `--no-verify`).
- Commit messages should follow the repository's existing style (check `git log --oneline -10`).
- Commit messages should focus on "why" not "what". Use conventional commit prefixes where appropriate (feat, fix, chore, docs, refactor, test).
- If the session made no changes (or all session changes are already committed), report that and exit.
- This skill intentionally ignores dirty files from before the session — use `/commit-all` to handle those.
