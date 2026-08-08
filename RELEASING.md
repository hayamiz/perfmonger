# Releasing PerfMonger

This is the maintainer runbook for cutting a PerfMonger release. Releases are
fully automated by GoReleaser running in GitHub Actions: pushing a version tag
triggers a gated build that produces prebuilt Linux binaries, a source archive,
and SHA256 checksums, and publishes them as a GitHub Release.

## Version scheme

There is a single source of truth for the version: the `VERSION` variable in
[`core/cmd/perfmonger/main.go`](core/cmd/perfmonger/main.go).

- It defaults to `"dev"`, so local `make build` binaries report
  `PerfMonger version dev` for `perfmonger --version`.
- At release time GoReleaser injects the tag (without the leading `v`) via
  `-ldflags "-X main.VERSION=<tag>"`. For example, tag `v1.0.0` produces a
  binary that reports `1.0.0`.

You do not edit `VERSION` by hand for a release — the tag drives it.

## Pre-release checklist

1. The working tree is clean and on the commit you intend to release.
2. All tests are green locally:
   ```sh
   make build && make vet && make test
   ```
   (`make build` must run before `make test`: the pytest suite expects the
   prebuilt binary at `lib/exec/perfmonger_linux_amd64`.)
3. The top entry of [`NEWS.md`](NEWS.md) describes this release with the correct
   version number and date. The release notes attached to the GitHub Release are
   extracted verbatim from this top section, so make sure it reads well.
4. (Optional but recommended) Run the local GoReleaser pre-flight below to catch
   config or build problems before pushing a tag.

## Local GoReleaser pre-flight (optional)

The release itself runs GoReleaser in CI, so you do not normally need it
locally. When changing `.goreleaser.yaml` (or before an important release) you
can validate the config and dry-run the build locally.

GoReleaser is a maintainer-only tool and is intentionally kept out of the
`core/go.mod` module graph (it pulls in a large dependency tree). Instead it is
installed on demand into your `GOBIN`, pinned via `GORELEASER_VERSION` in the
[`Makefile`](Makefile) to a version within the `~> v2` range CI uses.

```sh
# Recommended: install into the repo-local dev env tree (no sudo, isolated).
source 00_LOAD_GO_DEVENV.sh

make tools           # go install the pinned GoReleaser into GOBIN
make release-check   # goreleaser check + goreleaser release --snapshot --clean
```

`make release-check` depends on `make tools`, so a single `make release-check`
also installs GoReleaser if needed. The snapshot build writes to `dist/` (git
-ignored) and publishes nothing. On a host with an older Go, the default
`GOTOOLCHAIN=auto` lets `go install` fetch the toolchain GoReleaser requires.

To upgrade GoReleaser, bump the single `GORELEASER_VERSION` line in the
`Makefile` (keep it within `~> v2` to stay consistent with
[`.github/workflows/release.yml`](.github/workflows/release.yml)).

## Cutting a release

1. Update the top heading of `NEWS.md` to the new version and release date,
   e.g. `## 2026-05-29: PerfMonger 1.0.0`, and finalize the bullet list of
   changes.
2. Commit the `NEWS.md` change:
   ```sh
   git add NEWS.md
   git commit -m "Release vX.Y.Z"
   ```
3. Tag the release commit and push the tag:
   ```sh
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

Pushing a tag matching `v*.*.*` triggers the `Release` workflow
([`.github/workflows/release.yml`](.github/workflows/release.yml)).

## What the release workflow does automatically

On a `v*.*.*` tag push, `release.yml`:

1. Checks out the repository with full history (`fetch-depth: 0`, required by
   GoReleaser to compute the version).
2. Sets up Go 1.25 and `uv`.
3. Runs the test gate: `make build && make vet && make test`. If this fails the
   release is aborted.
4. Extracts the top section of `NEWS.md` into `/tmp/relnotes.md`:
   ```sh
   awk '/^## /{n++} n==1{print} n==2{exit}' NEWS.md > /tmp/relnotes.md
   ```
5. Runs `goreleaser release --clean --release-notes=/tmp/relnotes.md` (GoReleaser
   pinned to `~> v2`). GoReleaser:
   - cross-compiles the `perfmonger` binary for `linux/amd64` and `linux/arm64`
     with `CGO_ENABLED=0` (no cgo, so a single runner cross-builds both);
   - publishes each as a version-less tar.gz archive named
     `perfmonger_linux_amd64.tar.gz` / `perfmonger_linux_arm64.tar.gz`, each
     bundling the `perfmonger` binary plus `COPYING`, `README.md`, and
     `NEWS.md`;
   - generates a source code `tar.gz`;
   - generates a `checksums.txt` (SHA256) file;
   - creates the GitHub Release using the extracted `NEWS.md` notes (GoReleaser's
     own git-log changelog is disabled).

Note: GoReleaser's build output (`dist/`) is independent of `make build`'s output
(`lib/exec/`); they coexist.

## Verifying the published artifacts

After the workflow succeeds, on the
[Releases page](https://github.com/hayamiz/perfmonger/releases) confirm that the
release contains:

- `perfmonger_linux_amd64.tar.gz`
- `perfmonger_linux_arm64.tar.gz`
- a source archive (`*.tar.gz`)
- `checksums.txt`
- release notes matching the top `NEWS.md` entry.

Smoke-test a downloaded archive. The `releases/latest/download/` redirect
always resolves to the most recent release, so no version number is needed:

```sh
curl -L -o perfmonger_linux_amd64.tar.gz https://github.com/hayamiz/perfmonger/releases/latest/download/perfmonger_linux_amd64.tar.gz
tar xzf perfmonger_linux_amd64.tar.gz
./perfmonger --version   # should print: PerfMonger version X.Y.Z
```

Optionally verify the checksum against `checksums.txt`:

```sh
sha256sum -c checksums.txt
```
