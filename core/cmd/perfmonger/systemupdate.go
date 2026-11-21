package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// The self-update logic follows the approach of catscope's updater.go
// (download next to the running binary, restore the permission bits, then
// os.Rename over it) and adds what that implementation lacks: version
// comparison, SHA256 verification against the release's checksums.txt,
// release notes, a confirmation prompt, and a smoke test of the new binary
// before it replaces the running one.

const (
	// systemUpdateRepoURL is the GitHub repository whose releases are used.
	systemUpdateRepoURL = "https://github.com/hayamiz/perfmonger"

	// systemUpdateInstallDocURL points users at the manual installation
	// instructions when self-update is not possible.
	systemUpdateInstallDocURL = "https://github.com/hayamiz/perfmonger#installation"

	// checksumsAssetName is the GoReleaser checksum file attached to releases.
	checksumsAssetName = "checksums.txt"

	// updateTempPattern is the os.CreateTemp pattern of the file that holds the
	// new binary next to the running one until it is renamed into place.
	updateTempPattern = ".perfmonger-update-*"

	// smokeTestTimeout bounds how long the new binary may take to answer
	// --version.
	smokeTestTimeout = 30 * time.Second
)

// Default size limits applied while downloading and extracting a release.
const (
	defaultMaxChecksumsSize int64 = 1 << 20   // checksums.txt: 1 MiB
	defaultMaxTarballSize   int64 = 64 << 20  // release tar.gz: 64 MiB
	defaultMaxBinarySize    int64 = 128 << 20 // extracted binary: 128 MiB
	defaultMaxNewsSize      int64 = 4 << 20   // extracted NEWS.md: 4 MiB
)

var (
	// releaseVersionPattern matches a plain release version such as "1.1.0"
	// (the form release builds inject into VERSION).
	releaseVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

	// releaseTagPattern matches a release tag such as "v1.1.0". The tag is
	// embedded into download URLs, so nothing else is accepted.
	releaseTagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

	// newsHeadingPattern matches a NEWS.md release heading such as
	// "## 2026-11-21: PerfMonger 1.1.0".
	newsHeadingPattern = regexp.MustCompile(`^## \d{4}-\d{2}-\d{2}: PerfMonger (\d+\.\d+\.\d+)\s*$`)

	// versionOutputPattern extracts the version from `perfmonger --version`
	// output ("PerfMonger version 1.1.0").
	versionOutputPattern = regexp.MustCompile(`(?m)^PerfMonger version (\S+)\s*$`)

	// sha256HexPattern matches a lowercase or uppercase hex SHA256 digest.
	sha256HexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// releaseVersion is a numeric X.Y.Z version.
type releaseVersion struct {
	Major, Minor, Patch int
}

// parseReleaseVersion parses a plain "X.Y.Z" version. Anything else, including
// "dev", a leading "v", and pre-release suffixes such as "-rc1", is rejected.
func parseReleaseVersion(s string) (releaseVersion, error) {
	m := releaseVersionPattern.FindStringSubmatch(s)
	if m == nil {
		return releaseVersion{}, fmt.Errorf("invalid version %q: expected X.Y.Z", s)
	}
	var parts [3]int
	for i := range parts {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return releaseVersion{}, fmt.Errorf("invalid version %q: %v", s, err)
		}
		parts[i] = n
	}
	return releaseVersion{Major: parts[0], Minor: parts[1], Patch: parts[2]}, nil
}

// parseReleaseTag validates a release tag ("vX.Y.Z") and returns its version.
func parseReleaseTag(tag string) (releaseVersion, error) {
	if !releaseTagPattern.MatchString(tag) {
		return releaseVersion{}, fmt.Errorf("invalid release tag %q: expected vX.Y.Z", tag)
	}
	return parseReleaseVersion(strings.TrimPrefix(tag, "v"))
}

// String formats the version as "X.Y.Z".
func (v releaseVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0 or +1 when v is older than, equal to, or newer than o.
func (v releaseVersion) Compare(o releaseVersion) int {
	switch {
	case v.Major != o.Major:
		return compareInts(v.Major, o.Major)
	case v.Minor != o.Minor:
		return compareInts(v.Minor, o.Minor)
	default:
		return compareInts(v.Patch, o.Patch)
	}
}

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// updateLimits holds the size limits applied to downloaded and extracted data.
type updateLimits struct {
	ChecksumsSize int64
	TarballSize   int64
	BinarySize    int64
	NewsSize      int64
}

func defaultUpdateLimits() updateLimits {
	return updateLimits{
		ChecksumsSize: defaultMaxChecksumsSize,
		TarballSize:   defaultMaxTarballSize,
		BinarySize:    defaultMaxBinarySize,
		NewsSize:      defaultMaxNewsSize,
	}
}

// systemUpdater replaces the running perfmonger binary with the latest GitHub
// release. Every environment dependency is a field so that tests can drive it
// against an httptest server and a scratch directory.
type systemUpdater struct {
	// BaseURL is the repository URL, e.g. https://github.com/hayamiz/perfmonger.
	// Releases are resolved under <BaseURL>/releases/.
	BaseURL string
	// HTTPClient performs all requests. Downloads follow redirects with it; the
	// latest-release lookup uses a copy that does not follow redirects.
	HTTPClient *http.Client
	// ExePath is the path of the running binary (typically os.Executable()).
	ExePath string
	// CurrentVersion is the version of the running binary (main.VERSION).
	CurrentVersion string
	// GOOS and GOARCH select the release asset.
	GOOS   string
	GOARCH string
	// Stdin, Stdout and Stderr are used for the prompt, progress and warnings.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// IsTerminal reports whether Stdin is an interactive terminal.
	IsTerminal func() bool
	// CheckOnly reports the current and latest versions without updating.
	CheckOnly bool
	// AssumeYes skips the confirmation prompt.
	AssumeYes bool
	// Limits bounds the size of downloaded and extracted data.
	Limits updateLimits
}

// assetName maps the platform to the GoReleaser archive name.
func (u *systemUpdater) assetName() (string, error) {
	if u.GOOS != "linux" {
		return "", fmt.Errorf("system-update is not supported on %s/%s: release builds are published for linux/amd64 and linux/arm64 only", u.GOOS, u.GOARCH)
	}
	switch u.GOARCH {
	case "amd64", "arm64":
		return fmt.Sprintf("perfmonger_%s_%s.tar.gz", u.GOOS, u.GOARCH), nil
	default:
		return "", fmt.Errorf("system-update is not supported on %s/%s: release builds are published for linux/amd64 and linux/arm64 only", u.GOOS, u.GOARCH)
	}
}

// parseCurrentVersion validates the running binary's version. It is called
// before any network access so that development builds never get replaced.
func (u *systemUpdater) parseCurrentVersion() (releaseVersion, error) {
	if u.CurrentVersion == "dev" {
		return releaseVersion{}, fmt.Errorf("system-update is not available for development builds (version \"dev\"); install a release build manually, see the Installation section of README.md (%s)", systemUpdateInstallDocURL)
	}
	v, err := parseReleaseVersion(u.CurrentVersion)
	if err != nil {
		return releaseVersion{}, fmt.Errorf("system-update is not available for this build (version %q is not a release version X.Y.Z); install a release build manually, see the Installation section of README.md (%s)", u.CurrentVersion, systemUpdateInstallDocURL)
	}
	return v, nil
}

// newRequest builds a GET request carrying the perfmonger User-Agent.
func (u *systemUpdater) newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "perfmonger/"+u.CurrentVersion)
	return req, nil
}

// resolveLatestTag asks GitHub where /releases/latest redirects to, without
// following the redirect, and returns the tag at the end of the Location
// (".../releases/tag/vX.Y.Z"). The tag is validated strictly because it is
// embedded into the download URLs.
func (u *systemUpdater) resolveLatestTag(ctx context.Context) (string, releaseVersion, error) {
	latestURL := u.BaseURL + "/releases/latest"
	req, err := u.newRequest(ctx, latestURL)
	if err != nil {
		return "", releaseVersion{}, err
	}

	noRedirectClient := *u.HTTPClient
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return "", releaseVersion{}, fmt.Errorf("failed to look up the latest release: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
	default:
		return "", releaseVersion{}, fmt.Errorf("failed to look up the latest release: GET %s: expected a redirect, got HTTP status %s", latestURL, resp.Status)
	}

	location, err := resp.Location()
	if err != nil {
		return "", releaseVersion{}, fmt.Errorf("failed to look up the latest release: GET %s: %v", latestURL, err)
	}
	const tagMarker = "/releases/tag/"
	idx := strings.LastIndex(location.Path, tagMarker)
	if idx < 0 {
		return "", releaseVersion{}, fmt.Errorf("failed to look up the latest release: unexpected redirect to %q", location.String())
	}
	tag := location.Path[idx+len(tagMarker):]
	version, err := parseReleaseTag(tag)
	if err != nil {
		return "", releaseVersion{}, fmt.Errorf("failed to look up the latest release: %v", err)
	}
	return tag, version, nil
}

// download fetches url (following redirects) and returns at most limit bytes.
// Any status other than 200 and any body larger than limit is an error.
func (u *systemUpdater) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := u.newRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: HTTP status %s", url, resp.Status)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("failed to download %s: size %d exceeds the limit of %d bytes", url, resp.ContentLength, limit)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("failed to download %s: size exceeds the limit of %d bytes", url, limit)
	}
	return data, nil
}

// parseChecksums parses a GoReleaser/sha256sum style checksums.txt
// ("<sha256 hex>  <file name>" per line) and returns the digest for asset.
// Malformed lines, a malformed digest, duplicate entries for asset, and a
// missing entry are all errors.
func parseChecksums(data []byte, asset string) ([]byte, error) {
	var digest []byte
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !sha256HexPattern.MatchString(fields[0]) {
			return nil, fmt.Errorf("malformed line %d in %s", lineNo, checksumsAssetName)
		}
		// sha256sum marks binary-mode entries with a leading '*'.
		name := strings.TrimPrefix(fields[1], "*")
		if name != asset {
			continue
		}
		if digest != nil {
			return nil, fmt.Errorf("duplicate entry for %s in %s", asset, checksumsAssetName)
		}
		decoded, err := hex.DecodeString(fields[0])
		if err != nil {
			return nil, fmt.Errorf("malformed line %d in %s: %v", lineNo, checksumsAssetName, err)
		}
		digest = decoded
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", checksumsAssetName, err)
	}
	if digest == nil {
		return nil, fmt.Errorf("no checksum for %s in %s", asset, checksumsAssetName)
	}
	return digest, nil
}

// verifyChecksum checks the SHA256 of data against the checksums.txt entry for
// asset.
func verifyChecksum(data, checksums []byte, asset string) error {
	expected, err := parseChecksums(checksums, asset)
	if err != nil {
		return err
	}
	actual := sha256.Sum256(data)
	if !bytes.Equal(actual[:], expected) {
		return fmt.Errorf("SHA256 mismatch for %s: expected %x, got %x", asset, expected, actual[:])
	}
	return nil
}

// extractRelease reads a verified release tarball. It copies the regular file
// "perfmonger" (or "./perfmonger") into binaryOut and returns the content of
// NEWS.md if present. No other entry is ever written anywhere: entries with
// other names (including path-traversal names such as "../perfmonger") are
// skipped, and a "perfmonger" entry that is not a regular file is an error.
func extractRelease(tarball []byte, binaryOut io.Writer, limits updateLimits) (news []byte, err error) {
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		return nil, fmt.Errorf("failed to read release archive: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	foundBinary := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read release archive: %v", err)
		}

		switch strings.TrimPrefix(hdr.Name, "./") {
		case "perfmonger":
			if foundBinary {
				return nil, fmt.Errorf("release archive contains more than one perfmonger entry")
			}
			foundBinary = true
			if hdr.Typeflag != tar.TypeReg {
				return nil, fmt.Errorf("release archive entry %q is not a regular file", hdr.Name)
			}
			if hdr.Size > limits.BinarySize {
				return nil, fmt.Errorf("release archive entry %q is too large (%d bytes, limit %d)", hdr.Name, hdr.Size, limits.BinarySize)
			}
			n, err := io.Copy(binaryOut, io.LimitReader(tr, limits.BinarySize+1))
			if err != nil {
				return nil, fmt.Errorf("failed to extract %q: %v", hdr.Name, err)
			}
			if n > limits.BinarySize {
				return nil, fmt.Errorf("release archive entry %q is too large (limit %d bytes)", hdr.Name, limits.BinarySize)
			}
		case "NEWS.md":
			if hdr.Typeflag != tar.TypeReg || hdr.Size > limits.NewsSize || news != nil {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(tr, limits.NewsSize+1))
			if err != nil {
				return nil, fmt.Errorf("failed to extract %q: %v", hdr.Name, err)
			}
			if int64(len(data)) <= limits.NewsSize {
				news = data
			}
		}
	}
	if !foundBinary {
		return nil, fmt.Errorf("release archive does not contain a perfmonger binary")
	}
	return news, nil
}

// extractReleaseNotes returns the NEWS.md sections for versions v with
// current < v <= latest, in the order they appear in the file. Sections start
// at a "## YYYY-MM-DD: PerfMonger X.Y.Z" heading and end at the next level-2
// heading.
func extractReleaseNotes(news []byte, current, latest releaseVersion) string {
	var out strings.Builder
	including := false
	scanner := bufio.NewScanner(bytes.NewReader(news))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			including = false
			if m := newsHeadingPattern.FindStringSubmatch(line); m != nil {
				if v, err := parseReleaseVersion(m[1]); err == nil {
					including = v.Compare(current) > 0 && v.Compare(latest) <= 0
				}
			}
		}
		if including {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

// updateTarget describes the file that will be replaced.
type updateTarget struct {
	path string      // real path of the running binary (symlinks resolved)
	perm os.FileMode // permission bits to restore on the new binary
}

// resolveTarget resolves the real path of the running binary and records its
// permission bits.
func (u *systemUpdater) resolveTarget() (updateTarget, error) {
	if u.ExePath == "" {
		return updateTarget{}, fmt.Errorf("cannot determine the path of the running perfmonger binary")
	}
	realPath, err := filepath.EvalSymlinks(u.ExePath)
	if err != nil {
		return updateTarget{}, fmt.Errorf("cannot resolve the path of the running perfmonger binary %s: %v", u.ExePath, err)
	}
	realPath, err = filepath.Abs(realPath)
	if err != nil {
		return updateTarget{}, fmt.Errorf("cannot resolve the path of the running perfmonger binary %s: %v", u.ExePath, err)
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return updateTarget{}, fmt.Errorf("cannot stat %s: %v", realPath, err)
	}
	if !info.Mode().IsRegular() {
		return updateTarget{}, fmt.Errorf("cannot update %s: not a regular file", realPath)
	}
	return updateTarget{path: realPath, perm: info.Mode().Perm()}, nil
}

// confirm asks a yes/no question on Stdout and reads the answer from Stdin.
// Anything other than "y" or "yes" (including EOF) means no. The read happens
// in a goroutine so that a cancelled context (SIGINT/SIGTERM) aborts the wait.
func (u *systemUpdater) confirm(ctx context.Context, question string) (bool, error) {
	fmt.Fprintf(u.Stdout, "%s [y/N]: ", question)

	type answer struct {
		text string
		err  error
	}
	answers := make(chan answer, 1)
	go func() {
		text, err := bufio.NewReader(u.Stdin).ReadString('\n')
		answers <- answer{text: text, err: err}
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(u.Stdout)
		return false, ctx.Err()
	case a := <-answers:
		if a.err != nil && a.err != io.EOF {
			return false, fmt.Errorf("failed to read the answer: %v", a.err)
		}
		if a.err == io.EOF {
			fmt.Fprintln(u.Stdout)
		}
		switch strings.ToLower(strings.TrimSpace(a.text)) {
		case "y", "yes":
			return true, nil
		default:
			return false, nil
		}
	}
}

// smokeTest runs `<path> --version` and requires it to report want.
func smokeTest(ctx context.Context, path string, want releaseVersion) error {
	ctx, cancel := context.WithTimeout(ctx, smokeTestTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Errorf("the downloaded binary failed to run (%s --version): %v", path, err)
	}
	m := versionOutputPattern.FindSubmatch(out)
	if m == nil {
		return fmt.Errorf("the downloaded binary printed an unexpected version string: %q", strings.TrimSpace(string(out)))
	}
	if string(m[1]) != want.String() {
		return fmt.Errorf("the downloaded binary reports version %s, expected %s", m[1], want)
	}
	return nil
}

// syncDir fsyncs a directory so that a rename inside it is durable. Errors are
// ignored: the rename itself has already succeeded.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

// run performs the update. See the system-update section of
// doc/architecture.md for the overall flow.
func (u *systemUpdater) run(ctx context.Context) error {
	// 1-2. Environment and version checks, before any network access.
	asset, err := u.assetName()
	if err != nil {
		return err
	}
	current, err := u.parseCurrentVersion()
	if err != nil {
		return err
	}

	// 3. Resolve the latest release tag; every download below is pinned to it.
	tag, latest, err := u.resolveLatestTag(ctx)
	if err != nil {
		return err
	}

	// 4. Compare versions.
	fmt.Fprintf(u.Stdout, "Current version: %s\n", current)
	fmt.Fprintf(u.Stdout, "Latest version:  %s\n", latest)
	cmp := latest.Compare(current)
	if u.CheckOnly {
		if cmp > 0 {
			fmt.Fprintln(u.Stdout, "A newer version is available. Run 'perfmonger system-update' to update.")
		} else {
			fmt.Fprintln(u.Stdout, "perfmonger is up to date.")
		}
		return nil
	}
	if cmp <= 0 {
		fmt.Fprintln(u.Stdout, "perfmonger is up to date.")
		return nil
	}

	// 5. Make sure the binary can be replaced before downloading anything. The
	// temporary file created here receives the new binary and is removed on
	// every failure path.
	target, err := u.resolveTarget()
	if err != nil {
		return err
	}
	dir := filepath.Dir(target.path)
	tmp, err := os.CreateTemp(dir, updateTempPattern)
	if err != nil {
		return fmt.Errorf("cannot update %s: cannot create a file in %s (%v); if perfmonger is installed system-wide, re-run with sudo (e.g. sudo %s system-update)", target.path, dir, err, target.path)
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if !u.AssumeYes && !u.IsTerminal() {
		return fmt.Errorf("stdin is not a terminal, so the update cannot be confirmed; re-run with --yes to update without confirmation")
	}

	// 6. Download checksums.txt and the tarball, and verify the tarball before
	// looking inside it.
	downloadBase := u.BaseURL + "/releases/download/" + tag + "/"
	fmt.Fprintf(u.Stdout, "Downloading %s (%s)...\n", asset, tag)
	checksums, err := u.download(ctx, downloadBase+checksumsAssetName, u.Limits.ChecksumsSize)
	if err != nil {
		return err
	}
	tarball, err := u.download(ctx, downloadBase+asset, u.Limits.TarballSize)
	if err != nil {
		return err
	}
	if err := verifyChecksum(tarball, checksums, asset); err != nil {
		return err
	}
	fmt.Fprintf(u.Stdout, "Verified the SHA256 checksum of %s.\n", asset)

	// 7. Extract the binary into the temporary file and NEWS.md into memory.
	news, err := extractRelease(tarball, tmp, u.Limits)
	if err != nil {
		return err
	}

	// 8. Show the release notes between the current and latest versions.
	if news == nil {
		fmt.Fprintln(u.Stderr, "warning: NEWS.md is missing from the release archive; release notes are not available")
	} else if notes := extractReleaseNotes(news, current, latest); notes != "" {
		fmt.Fprintf(u.Stdout, "\nRelease notes:\n\n%s\n\n", notes)
	} else {
		fmt.Fprintf(u.Stderr, "warning: NEWS.md has no release notes for versions after %s\n", current)
	}

	// 9. Ask for confirmation.
	if !u.AssumeYes {
		ok, err := u.confirm(ctx, fmt.Sprintf("Update perfmonger %s -> %s (%s)?", current, latest, target.path))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(u.Stdout, "Update cancelled.")
			return nil
		}
	}

	// 10. Persist the new binary, restore the permission bits, make sure it
	// runs and reports the expected version, then rename it into place.
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to write %s: %v", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %v", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, target.perm); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %v", tmpPath, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := smokeTest(ctx, tmpPath, latest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, target.path); err != nil {
		return fmt.Errorf("failed to replace %s: %v", target.path, err)
	}
	renamed = true
	syncDir(dir)

	fmt.Fprintf(u.Stdout, "Updated perfmonger %s -> %s (%s)\n", current, latest, target.path)
	return nil
}

// newUpdateHTTPClient returns the HTTP client used for production updates. It
// bounds every phase of a request and refuses redirects that downgrade an
// https request to plain http.
func newUpdateHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = 15 * time.Second
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{
		Transport: transport,
		Timeout:   5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
				return fmt.Errorf("refusing to follow a redirect from https to %s", req.URL.Scheme)
			}
			return nil
		},
	}
}

// isTerminalReader reports whether r is a terminal file.
func isTerminalReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// newSystemUpdateCommand creates the system-update subcommand and wires the
// updater to the real environment.
func newSystemUpdateCommand() *cobra.Command {
	var checkOnly, assumeYes bool

	cmd := &cobra.Command{
		Use:   "system-update",
		Short: "Update perfmonger to the latest release",
		Long: `Update the running perfmonger binary in place to the latest release
published on GitHub (https://github.com/hayamiz/perfmonger/releases).

The release archive is verified against the release's checksums.txt (SHA256)
before it is extracted, the release notes since the current version are shown,
and the binary is replaced only after confirmation. Replacing the binary needs
write permission to the directory that contains it; use sudo for system-wide
installations. Development builds (version "dev") cannot be updated.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Errors from here on are runtime failures, not usage errors.
			cmd.SilenceUsage = true

			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
			defer stop()

			exePath, err := os.Executable()
			if err != nil {
				exePath = ""
			}
			stdin := cmd.InOrStdin()
			updater := &systemUpdater{
				BaseURL:        systemUpdateRepoURL,
				HTTPClient:     newUpdateHTTPClient(),
				ExePath:        exePath,
				CurrentVersion: VERSION,
				GOOS:           runtime.GOOS,
				GOARCH:         runtime.GOARCH,
				Stdin:          stdin,
				Stdout:         cmd.OutOrStdout(),
				Stderr:         cmd.ErrOrStderr(),
				IsTerminal:     func() bool { return isTerminalReader(stdin) },
				CheckOnly:      checkOnly,
				AssumeYes:      assumeYes,
				Limits:         defaultUpdateLimits(),
			}
			if err := updater.run(ctx); err != nil {
				if ctx.Err() != nil {
					return fmt.Errorf("system-update interrupted; the installed binary was not changed")
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only report the current and latest versions; do not update")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Update without asking for confirmation")
	cmd.SetUsageTemplate(subCommandUsageTemplate)

	return cmd
}
