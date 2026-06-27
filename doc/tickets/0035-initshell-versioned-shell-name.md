---
title: SHELL env fallback fails for versioned shell binary names like "bash-5.1"
type: bug
priority: low
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/cmd/perfmonger/initshell.go` around lines 126-128, when `ps` fails,
`getParentShell()` falls back to `filepath.Base(os.Getenv("SHELL"))`. This
correctly strips directory prefixes (`/bin/bash` -> `bash`), but on systems where
the shell binary is named with a version suffix — e.g. `/usr/bin/bash-5.1` or
`/usr/local/bin/zsh-5.9` — `filepath.Base` returns `bash-5.1` or `zsh-5.9`.

Neither matches `bash` or `zsh` in the switch in `opts.run()`, so the command
fails with an "unsupported shell" error even though the user is running a supported
shell.

Why it is a bug: a supported shell is rejected purely because of a version-suffixed
binary name.

Suggested direction: normalize the detected shell name before matching — e.g.
match on a prefix (`strings.HasPrefix(name, "bash")` / `"zsh"`) or strip a trailing
`-<version>` suffix — so versioned binary names are recognized.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: SHELL-env fallback: filepath.Base("/usr/bin/bash-5.1") returns "bash-5.1", not matching the switch. Fix: normalize the shell name (prefix match or strip trailing -<version>) before the switch.
