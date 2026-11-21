"""Regression test for #0068: the GitHub Release body must not be empty.

`changelog.disable: true` makes goreleaser skip the changelog pipe, which is
also where the --release-notes file is loaded.
"""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def changelog_is_disabled(text):
    """Return True if a top-level `changelog:` block sets `disable: true`."""
    in_block = False
    for line in text.splitlines():
        if re.match(r"^\S", line) and not line.startswith("#"):
            in_block = re.match(r"^changelog:\s*(#.*)?$", line) is not None
            continue
        if in_block and re.match(r"^\s+disable:\s*true\b", line):
            return True
    return False


def test_checker_detects_old_config():
    assert changelog_is_disabled("changelog:\n  disable: true\n")
    assert not changelog_is_disabled("source:\n  enabled: true\n")


def test_release_workflow_passes_release_notes():
    text = (ROOT / ".github" / "workflows" / "release.yml").read_text()
    assert re.search(r"goreleaser|args:\s*release", text)
    assert "--release-notes" in text


def test_goreleaser_does_not_disable_changelog():
    text = (ROOT / ".goreleaser.yaml").read_text()
    assert not changelog_is_disabled(text), (
        "changelog.disable: true skips the --release-notes file too (#0068)"
    )
