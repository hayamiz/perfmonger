---
name: commit-all
description: "Commit all dirty files in the worktree, grouped into semantically coherent commits. Excludes temporary artifacts and scratch notes."
argument-hint: "[--dry-run] [--include-untracked]"
allowed-tools: Bash(*) Read Glob Grep
---

# Commit All

Commit all uncommitted changes in the worktree, organized into semantically coherent groups. Each group becomes a separate commit with a descriptive message.

## Arguments

Parse `$ARGUMENTS` for:
- `--dry-run` (optional): Show the planned commit groups without actually committing. Default: false.
- `--include-untracked` (optional): Include untracked files even if they look temporary. Default: false (temporary files are excluded).

## Execution Flow

### Step 1: Inventory

Run `git status --porcelain` to collect all dirty files (modified, added, deleted, untracked).

Display the full list to the user.

### Step 2: Classify and Exclude Temporaries

Exclude files that appear to be temporary artifacts or scratch notes unless `--include-untracked` is set. Exclusion criteria:

1. **Gitignored files** — already excluded by git status, but double-check with `git status --ignored --porcelain` if needed.
2. **Scratch / memo files** — files matching patterns like:
   - `*.tmp`, `*.bak`, `*.swp`, `*.swo`, `*~`
   - `scratch.*`, `memo.*`, `notes.*`, `TODO.txt`, `todo.txt`
   - Files in directories named `tmp/`, `temp/`, `scratch/`, `.scratch/`
3. **Build / generated artifacts** — files matching patterns like:
   - `__pycache__/`, `*.pyc`, `*.pyo`
   - `node_modules/`, `dist/`, `build/`, `.cache/`
   - `*.log` (unless in a committed `logs/` directory structure)
4. **Large binary files** — files over 10MB that are not already tracked by git.

For each excluded file, note the reason. Present the exclusion list to the user.

If any exclusion seems questionable (e.g., a file with a real extension like `.md` or `.sh` in a scratch-like directory), flag it explicitly and ask the user.

### Step 3: Semantic Grouping

Group the remaining files into semantically coherent commits. Criteria for grouping:

1. **By functional area**: Files that belong to the same feature, module, or subsystem go together.
   - Example: a source file and its corresponding test → one commit
   - Example: related config/prompt files in the same subsystem → one commit if related
2. **By change type**: If files span multiple areas but share a change type, group by type:
   - Documentation updates
   - Configuration changes
   - Test changes
   - Source code changes
   - Other category changes (refer to CLAUDE.md for project-specific directory conventions)
3. **By dependency**: If file A depends on file B's changes (e.g., a new function used by a new caller), they go in the same commit or A's commit comes after B's.

Use these heuristics to form groups, then determine commit order (dependencies first).

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

...

### Excluded (temporary/generated)
  - path/to/temp (reason)
```

### Step 4: Confirm

If `--dry-run`, stop here.

Otherwise, ask the user to confirm the plan. Accept adjustments (move files between groups, change messages, include/exclude files).

### Step 5: Execute Commits

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

### Step 6: Summary

Report:
```
## Commit Summary

| # | Commit | Files | Hash |
|---|--------|-------|------|
| 1 | <message> | N files | abc1234 |
| 2 | <message> | N files | def5678 |
...

Total: N commits, M files committed
Excluded: K files (temporary/generated)
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
