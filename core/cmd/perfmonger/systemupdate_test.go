package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

const testAssetName = "perfmonger_linux_amd64.tar.gz"

// testNews mimics the NEWS.md layout shipped in release archives.
const testNews = `## 2026-12-01: PerfMonger 1.3.0
  * Notes for 1.3.0

## 2026-11-30: PerfMonger 1.2.0
  * Notes for 1.2.0

## 2026-11-21: PerfMonger 1.1.0
  * Notes for 1.1.0

## 2026-05-29: PerfMonger 1.0.0
  * Notes for 1.0.0
`

// fakeBinaryScript returns a shell script that behaves like
// `perfmonger --version` of the given version, so that the smoke test can run
// the "downloaded binary".
func fakeBinaryScript(version string) []byte {
	return []byte("#!/bin/sh\necho 'PerfMonger version " + version + "'\n")
}

// tarEntry describes one entry of a test release archive.
type tarEntry struct {
	Name     string
	Typeflag byte // defaults to tar.TypeReg
	Body     []byte
	Linkname string
	Mode     int64 // defaults to 0755
}

// buildTarball builds a gzip-compressed tar archive from entries.
func buildTarball(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typeflag := e.Typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		mode := e.Mode
		if mode == 0 {
			mode = 0o755
		}
		hdr := &tar.Header{Name: e.Name, Typeflag: typeflag, Mode: mode, Linkname: e.Linkname}
		if typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.Body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar WriteHeader(%q): %v", e.Name, err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tw.Write(e.Body); err != nil {
				t.Fatalf("tar Write(%q): %v", e.Name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close: %v", err)
	}
	return buf.Bytes()
}

// checksumLine formats a GoReleaser checksums.txt line for data.
func checksumLine(data []byte, name string) string {
	return fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
}

// standardEntries returns the entries of a well-formed release archive.
func standardEntries(version string) []tarEntry {
	return []tarEntry{
		{Name: "COPYING", Body: []byte("license text\n"), Mode: 0o644},
		{Name: "NEWS.md", Body: []byte(testNews), Mode: 0o644},
		{Name: "README.md", Body: []byte("readme\n"), Mode: 0o644},
		{Name: "perfmonger", Body: fakeBinaryScript(version)},
	}
}

// releaseAssets returns the release assets (tarball plus a checksums.txt that
// matches it) for an archive made of entries.
func releaseAssets(t *testing.T, entries []tarEntry) map[string][]byte {
	t.Helper()
	tarball := buildTarball(t, entries)
	checksums := checksumLine([]byte("arm64 tarball"), "perfmonger_linux_arm64.tar.gz") +
		checksumLine(tarball, testAssetName) +
		checksumLine([]byte("source"), "perfmonger_1.2.0.tar.gz")
	return map[string][]byte{
		testAssetName:      tarball,
		checksumsAssetName: []byte(checksums),
	}
}

// fakeReleaseServer imitates the GitHub release endpoints used by the
// updater: /releases/latest redirects to /releases/tag/<tag>, and
// /releases/download/<tag>/<asset> redirects to another path that serves the
// asset (GitHub redirects asset downloads to a different host).
type fakeReleaseServer struct {
	server *httptest.Server

	mu             sync.Mutex
	latestStatus   int
	latestLocation string // empty means no Location header
	tag            string
	assets         map[string][]byte
	paths          []string
	userAgents     []string
}

func newFakeReleaseServer(t *testing.T, tag string, assets map[string][]byte) *fakeReleaseServer {
	t.Helper()
	f := &fakeReleaseServer{
		latestStatus:   http.StatusFound,
		latestLocation: "/releases/tag/" + tag,
		tag:            tag,
		assets:         assets,
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeReleaseServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.userAgents = append(f.userAgents, r.UserAgent())
	status, location, tag := f.latestStatus, f.latestLocation, f.tag
	f.mu.Unlock()

	switch {
	case r.URL.Path == "/releases/latest":
		if location != "" {
			w.Header().Set("Location", location)
		}
		w.WriteHeader(status)
	case strings.HasPrefix(r.URL.Path, "/releases/download/"):
		http.Redirect(w, r, "/assets/"+strings.TrimPrefix(r.URL.Path, "/releases/download/"), http.StatusFound)
	case strings.HasPrefix(r.URL.Path, "/assets/"):
		assetTag, name, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/assets/"), "/")
		data, found := f.assets[name]
		if !ok || assetTag != tag || !found {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	default:
		http.NotFound(w, r)
	}
}

// requestCount returns how many requests hit a path with the given prefix.
func (f *fakeReleaseServer) requestCount(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, p := range f.paths {
		if strings.HasPrefix(p, prefix) {
			n++
		}
	}
	return n
}

// downloadCount returns how many asset download requests were made.
func (f *fakeReleaseServer) downloadCount() int {
	return f.requestCount("/releases/download/") + f.requestCount("/assets/")
}

// tarballCount returns how many requests asked for the release tarball.
func (f *fakeReleaseServer) tarballCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, p := range f.paths {
		if strings.HasSuffix(p, "/"+testAssetName) {
			n++
		}
	}
	return n
}

// installFakeBinary writes a fake installed perfmonger of the given version.
func installFakeBinary(t *testing.T, dir, version string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, "perfmonger")
	if err := os.WriteFile(path, fakeBinaryScript(version), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
	return path
}

// testUpdater bundles an updater with its captured output.
type testUpdater struct {
	*systemUpdater
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newTestUpdater(srv *fakeReleaseServer, exePath, current string) *testUpdater {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	u := &systemUpdater{
		BaseURL:        srv.server.URL,
		HTTPClient:     srv.server.Client(),
		ExePath:        exePath,
		CurrentVersion: current,
		GOOS:           "linux",
		GOARCH:         "amd64",
		Stdin:          strings.NewReader(""),
		Stdout:         stdout,
		Stderr:         stderr,
		IsTerminal:     func() bool { return false },
		Limits:         defaultUpdateLimits(),
	}
	return &testUpdater{systemUpdater: u, stdout: stdout, stderr: stderr}
}

// answerPrompt makes the updater behave as if a user typed answer on a TTY.
func (tu *testUpdater) answerPrompt(answer string) {
	tu.Stdin = strings.NewReader(answer)
	tu.IsTerminal = func() bool { return true }
}

func assertFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("content of %s = %q, want %q", path, got, want)
	}
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s): %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("mode of %s = %o, want %o", path, got, want)
	}
}

// assertNoTempFiles checks that no update temp file was left behind in dir.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, updateTempPattern))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("leftover temp files in %s: %v", dir, matches)
	}
}

// dirEntryNames lists the names in dir, sorted.
func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func assertErrorContains(t *testing.T, err error, substrings ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", substrings)
	}
	for _, s := range substrings {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err.Error(), s)
		}
	}
}

// ---------------------------------------------------------------------------
// Version parsing and comparison
// ---------------------------------------------------------------------------

func TestParseReleaseVersion(t *testing.T) {
	tests := []struct {
		in      string
		want    releaseVersion
		wantErr bool
	}{
		{in: "1.1.0", want: releaseVersion{1, 1, 0}},
		{in: "0.0.0", want: releaseVersion{0, 0, 0}},
		{in: "10.20.30", want: releaseVersion{10, 20, 30}},
		{in: "dev", wantErr: true},
		{in: "", wantErr: true},
		{in: "1.2", wantErr: true},
		{in: "1.2.3.4", wantErr: true},
		{in: "v1.2.3", wantErr: true},
		{in: "1.2.3-rc1", wantErr: true},
		{in: "1.2.3+build", wantErr: true},
		{in: "1.2.x", wantErr: true},
		{in: " 1.2.3", wantErr: true},
		{in: "1.2.3\n", wantErr: true},
		{in: "+1.2.3", wantErr: true},
		{in: "99999999999999999999.0.0", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.in), func(t *testing.T) {
			got, err := parseReleaseVersion(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseReleaseVersion(%q) = %v, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseReleaseVersion(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("parseReleaseVersion(%q) = %v, want %v", tt.in, got, tt.want)
			}
			if got.String() != tt.in {
				t.Errorf("String() = %q, want %q", got.String(), tt.in)
			}
		})
	}
}

func TestReleaseVersionCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.1.0", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.1.10", "1.1.9", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.1.0", "1.1.0", 0},
		{"1.0.0", "1.1.0", -1},
		{"0.14.4", "1.0.0", -1},
		{"1.1.0", "1.1.1", -1},
	}
	for _, tt := range tests {
		a, err := parseReleaseVersion(tt.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := parseReleaseVersion(tt.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Compare(b); got != tt.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseReleaseTag(t *testing.T) {
	valid := map[string]releaseVersion{
		"v1.2.3":  {1, 2, 3},
		"v10.0.1": {10, 0, 1},
	}
	for tag, want := range valid {
		got, err := parseReleaseTag(tag)
		if err != nil || got != want {
			t.Errorf("parseReleaseTag(%q) = %v, %v; want %v", tag, got, err, want)
		}
	}
	invalid := []string{
		"1.2.3", "v1.2", "v1.2.3-rc1", "v1.2.3/../../x", "v1.2.3/", "../v1.2.3",
		"V1.2.3", "v1.2.3 ", "", "latest",
	}
	for _, tag := range invalid {
		if got, err := parseReleaseTag(tag); err == nil {
			t.Errorf("parseReleaseTag(%q) = %v, want an error", tag, got)
		}
	}
}

func TestSystemUpdaterParseCurrentVersion(t *testing.T) {
	u := &systemUpdater{CurrentVersion: "dev"}
	_, err := u.parseCurrentVersion()
	assertErrorContains(t, err, `"dev"`, "Installation", "README")

	for _, v := range []string{"", "1.2.0-rc1", "v1.2.0", "1.2", "unknown"} {
		u := &systemUpdater{CurrentVersion: v}
		_, err := u.parseCurrentVersion()
		assertErrorContains(t, err, "install a release build manually")
	}

	u = &systemUpdater{CurrentVersion: "1.2.0"}
	got, err := u.parseCurrentVersion()
	if err != nil || got != (releaseVersion{1, 2, 0}) {
		t.Errorf("parseCurrentVersion() = %v, %v; want 1.2.0", got, err)
	}
}

func TestSystemUpdaterAssetName(t *testing.T) {
	supported := map[[2]string]string{
		{"linux", "amd64"}: "perfmonger_linux_amd64.tar.gz",
		{"linux", "arm64"}: "perfmonger_linux_arm64.tar.gz",
	}
	for platform, want := range supported {
		u := &systemUpdater{GOOS: platform[0], GOARCH: platform[1]}
		got, err := u.assetName()
		if err != nil || got != want {
			t.Errorf("assetName(%v) = %q, %v; want %q", platform, got, err, want)
		}
	}
	for _, platform := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "386"}, {"linux", "riscv64"}, {"windows", "amd64"}} {
		u := &systemUpdater{GOOS: platform[0], GOARCH: platform[1]}
		_, err := u.assetName()
		assertErrorContains(t, err, "not supported", platform[0]+"/"+platform[1])
	}
}

// ---------------------------------------------------------------------------
// Checksums
// ---------------------------------------------------------------------------

func TestParseChecksums(t *testing.T) {
	data := []byte("tarball bytes")
	sum := sha256.Sum256(data)
	hexSum := fmt.Sprintf("%x", sum)
	other := strings.Repeat("ab", 32)

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "goreleaser format", content: other + "  perfmonger_linux_arm64.tar.gz\n" + hexSum + "  " + testAssetName + "\n"},
		{name: "binary mode marker", content: hexSum + " *" + testAssetName + "\n"},
		{name: "uppercase hex", content: strings.ToUpper(hexSum) + "  " + testAssetName + "\n"},
		{name: "blank lines and no trailing newline", content: "\n" + other + "  a.tar.gz\n\n" + hexSum + "  " + testAssetName},
		{name: "missing entry", content: other + "  perfmonger_linux_arm64.tar.gz\n", wantErr: "no checksum for " + testAssetName},
		{name: "empty file", content: "", wantErr: "no checksum"},
		{name: "similar name only", content: hexSum + "  x" + testAssetName + "\n", wantErr: "no checksum"},
		{name: "one field", content: hexSum + "\n", wantErr: "malformed line 1"},
		{name: "three fields", content: hexSum + "  " + testAssetName + " extra\n", wantErr: "malformed line 1"},
		{name: "short digest", content: hexSum[:62] + "  " + testAssetName + "\n", wantErr: "malformed line 1"},
		{name: "non-hex digest", content: strings.Repeat("zz", 32) + "  " + testAssetName + "\n", wantErr: "malformed line 1"},
		{name: "malformed unrelated line", content: hexSum + "  " + testAssetName + "\ngarbage\n", wantErr: "malformed line 2"},
		{name: "duplicate entry", content: hexSum + "  " + testAssetName + "\n" + other + "  " + testAssetName + "\n", wantErr: "duplicate entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChecksums([]byte(tt.content), testAssetName)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(got, sum[:]) {
				t.Errorf("digest = %x, want %x", got, sum)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("tarball bytes")
	checksums := []byte(checksumLine(data, testAssetName))
	if err := verifyChecksum(data, checksums, testAssetName); err != nil {
		t.Errorf("verifyChecksum() with a matching digest: %v", err)
	}
	err := verifyChecksum([]byte("tampered bytes"), checksums, testAssetName)
	assertErrorContains(t, err, "SHA256 mismatch", testAssetName)
}

// ---------------------------------------------------------------------------
// Archive extraction
// ---------------------------------------------------------------------------

func TestExtractRelease(t *testing.T) {
	binary := fakeBinaryScript("1.2.0")
	news := []byte(testNews)

	tests := []struct {
		name       string
		entries    []tarEntry
		limits     func(*updateLimits)
		wantErr    string
		wantNews   []byte
		wantBinary []byte
	}{
		{
			name:       "standard archive",
			entries:    standardEntries("1.2.0"),
			wantNews:   news,
			wantBinary: binary,
		},
		{
			name:       "dot-slash names",
			entries:    []tarEntry{{Name: "./NEWS.md", Body: news}, {Name: "./perfmonger", Body: binary}},
			wantNews:   news,
			wantBinary: binary,
		},
		{
			name:       "missing NEWS.md",
			entries:    []tarEntry{{Name: "perfmonger", Body: binary}},
			wantBinary: binary,
		},
		{
			name:       "oversized NEWS.md is ignored",
			entries:    []tarEntry{{Name: "NEWS.md", Body: news}, {Name: "perfmonger", Body: binary}},
			limits:     func(l *updateLimits) { l.NewsSize = 8 },
			wantBinary: binary,
		},
		{
			name:    "no perfmonger entry",
			entries: []tarEntry{{Name: "NEWS.md", Body: news}, {Name: "README.md", Body: []byte("x")}},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name:    "path traversal name only",
			entries: []tarEntry{{Name: "../perfmonger", Body: binary}},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name:    "nested name only",
			entries: []tarEntry{{Name: "dist/perfmonger", Body: binary}},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name:    "absolute name only",
			entries: []tarEntry{{Name: "/perfmonger", Body: binary}},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name:    "symlink named perfmonger",
			entries: []tarEntry{{Name: "perfmonger", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}},
			wantErr: "not a regular file",
		},
		{
			name:    "hard link named perfmonger",
			entries: []tarEntry{{Name: "perfmonger", Typeflag: tar.TypeLink, Linkname: "README.md"}},
			wantErr: "not a regular file",
		},
		{
			name:    "directory named perfmonger",
			entries: []tarEntry{{Name: "perfmonger", Typeflag: tar.TypeDir}},
			wantErr: "not a regular file",
		},
		{
			name:    "directory named perfmonger/",
			entries: []tarEntry{{Name: "perfmonger/", Typeflag: tar.TypeDir}},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name:    "oversized binary",
			entries: []tarEntry{{Name: "perfmonger", Body: binary}},
			limits:  func(l *updateLimits) { l.BinarySize = int64(len(binary)) - 1 },
			wantErr: "too large",
		},
		{
			name:    "duplicate binary entries",
			entries: []tarEntry{{Name: "perfmonger", Body: binary}, {Name: "./perfmonger", Body: binary}},
			wantErr: "more than one perfmonger entry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := defaultUpdateLimits()
			if tt.limits != nil {
				tt.limits(&limits)
			}
			var out bytes.Buffer
			gotNews, err := extractRelease(buildTarball(t, tt.entries), &out, limits)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(gotNews, tt.wantNews) {
				t.Errorf("news = %q, want %q", gotNews, tt.wantNews)
			}
			if !bytes.Equal(out.Bytes(), tt.wantBinary) {
				t.Errorf("binary = %q, want %q", out.Bytes(), tt.wantBinary)
			}
		})
	}

	t.Run("not a gzip stream", func(t *testing.T) {
		_, err := extractRelease([]byte("not gzip"), io.Discard, defaultUpdateLimits())
		assertErrorContains(t, err, "failed to read release archive")
	})

	t.Run("header claims an oversized entry", func(t *testing.T) {
		// A header that declares a huge size must be rejected from the header
		// alone, without reading (or needing) the body.
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "perfmonger", Typeflag: tar.TypeReg, Mode: 0o755, Size: defaultMaxBinarySize + 1}); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		_, err := extractRelease(buf.Bytes(), &out, defaultUpdateLimits())
		assertErrorContains(t, err, "too large")
		if out.Len() != 0 {
			t.Errorf("wrote %d bytes for an oversized entry", out.Len())
		}
	})
}

// ---------------------------------------------------------------------------
// Release notes
// ---------------------------------------------------------------------------

func TestExtractReleaseNotes(t *testing.T) {
	v := func(s string) releaseVersion {
		t.Helper()
		r, err := parseReleaseVersion(s)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	t.Run("range between current and latest", func(t *testing.T) {
		notes := extractReleaseNotes([]byte(testNews), v("1.0.0"), v("1.2.0"))
		want := "## 2026-11-30: PerfMonger 1.2.0\n  * Notes for 1.2.0\n\n## 2026-11-21: PerfMonger 1.1.0\n  * Notes for 1.1.0"
		if notes != want {
			t.Errorf("notes = %q, want %q", notes, want)
		}
	})

	t.Run("no newer section", func(t *testing.T) {
		if notes := extractReleaseNotes([]byte(testNews), v("1.3.0"), v("1.3.0")); notes != "" {
			t.Errorf("notes = %q, want empty", notes)
		}
	})

	t.Run("other level-2 headings end a section", func(t *testing.T) {
		news := "## Unreleased\n  * not yet\n\n## 2026-11-30: PerfMonger 1.2.0\n  * Notes for 1.2.0\n\n## Misc\n  * unrelated\n"
		notes := extractReleaseNotes([]byte(news), v("1.1.0"), v("1.2.0"))
		if notes != "## 2026-11-30: PerfMonger 1.2.0\n  * Notes for 1.2.0" {
			t.Errorf("notes = %q", notes)
		}
	})

	t.Run("repository NEWS.md", func(t *testing.T) {
		news, err := os.ReadFile(filepath.Join("..", "..", "..", "NEWS.md"))
		if err != nil {
			t.Skipf("NEWS.md not readable: %v", err)
		}
		notes := extractReleaseNotes(news, v("1.0.0"), v("1.1.0"))
		if !strings.HasPrefix(notes, "## 2026-11-21: PerfMonger 1.1.0\n") {
			t.Errorf("notes do not start with the 1.1.0 heading: %q", notes[:min(len(notes), 80)])
		}
		if strings.Contains(notes, "PerfMonger 1.0.0") {
			t.Error("notes must not include the 1.0.0 section")
		}
	})
}

// ---------------------------------------------------------------------------
// Latest release resolution
// ---------------------------------------------------------------------------

func TestResolveLatestTag(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		location string
		wantTag  string
		wantErr  string
	}{
		{name: "relative location", status: http.StatusFound, location: "/releases/tag/v1.2.3", wantTag: "v1.2.3"},
		{name: "absolute location", status: http.StatusFound, location: "https://github.com/hayamiz/perfmonger/releases/tag/v1.2.3", wantTag: "v1.2.3"},
		{name: "301 redirect", status: http.StatusMovedPermanently, location: "/releases/tag/v2.0.0", wantTag: "v2.0.0"},
		{name: "query string is ignored", status: http.StatusFound, location: "/releases/tag/v1.2.3?from=latest", wantTag: "v1.2.3"},
		{name: "path traversal", status: http.StatusFound, location: "/releases/tag/v1.2.3/../../x", wantErr: "unexpected redirect"},
		{name: "absolute path traversal", status: http.StatusFound, location: "https://github.com/hayamiz/perfmonger/releases/tag/v1.2.3/../../x", wantErr: "unexpected redirect"},
		{name: "encoded slashes", status: http.StatusFound, location: "/releases/tag/v1.2.3%2F..%2F..%2Fx", wantErr: "invalid release tag"},
		{name: "trailing path", status: http.StatusFound, location: "/releases/tag/v1.2.3/extra", wantErr: "invalid release tag"},
		{name: "pre-release tag", status: http.StatusFound, location: "/releases/tag/v1.2.3-rc1", wantErr: "invalid release tag"},
		{name: "tag without v", status: http.StatusFound, location: "/releases/tag/1.2.3", wantErr: "invalid release tag"},
		{name: "not a tag URL", status: http.StatusFound, location: "/releases", wantErr: "unexpected redirect"},
		{name: "missing location", status: http.StatusFound, location: "", wantErr: "no Location header"},
		{name: "200 instead of redirect", status: http.StatusOK, location: "/releases/tag/v1.2.3", wantErr: "200"},
		{name: "404 (no releases)", status: http.StatusNotFound, location: "", wantErr: "404"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.3", nil)
			srv.latestStatus = tt.status
			srv.latestLocation = tt.location
			u := newTestUpdater(srv, "", "1.0.0")

			tag, version, err := u.resolveLatestTag(context.Background())
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", tag, tt.wantTag)
			}
			if "v"+version.String() != tt.wantTag {
				t.Errorf("version = %v, want %s", version, tt.wantTag)
			}
			// The redirect must not be followed: the tag page itself is
			// never requested.
			if n := srv.requestCount("/releases/tag/"); n != 0 {
				t.Errorf("the redirect was followed (%d requests to /releases/tag/)", n)
			}
			if got := srv.userAgents[0]; got != "perfmonger/1.0.0" {
				t.Errorf("User-Agent = %q, want %q", got, "perfmonger/1.0.0")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// End-to-end updater runs against the fake release server
// ---------------------------------------------------------------------------

func TestSystemUpdateReplacesBinary(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o751)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.answerPrompt("y\n")

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run() error: %v\nstdout:\n%s", err, u.stdout)
	}

	assertFileContent(t, exe, fakeBinaryScript("1.2.0"))
	assertFileMode(t, exe, 0o751)
	assertNoTempFiles(t, dir)
	if names := dirEntryNames(t, dir); len(names) != 1 || names[0] != "perfmonger" {
		t.Errorf("directory entries = %v, want [perfmonger]", names)
	}

	out := u.stdout.String()
	for _, want := range []string{
		"Current version: 1.0.0",
		"Latest version:  1.2.0",
		"Verified the SHA256 checksum",
		"## 2026-11-30: PerfMonger 1.2.0",
		"## 2026-11-21: PerfMonger 1.1.0",
		"Update perfmonger 1.0.0 -> 1.2.0 (" + exe + ")? [y/N]",
		"Updated perfmonger 1.0.0 -> 1.2.0",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout does not contain %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"PerfMonger 1.3.0", "PerfMonger 1.0.0\n"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("stdout unexpectedly contains %q:\n%s", unwanted, out)
		}
	}
	if u.stderr.Len() != 0 {
		t.Errorf("unexpected stderr output: %s", u.stderr)
	}

	// Downloads are pinned to the resolved tag and follow the asset redirect.
	for _, path := range []string{
		"/releases/download/v1.2.0/" + testAssetName,
		"/releases/download/v1.2.0/" + checksumsAssetName,
		"/assets/v1.2.0/" + testAssetName,
		"/assets/v1.2.0/" + checksumsAssetName,
	} {
		if srv.requestCount(path) != 1 {
			t.Errorf("expected exactly one request to %s, got %d", path, srv.requestCount(path))
		}
	}
	for _, ua := range srv.userAgents {
		if ua != "perfmonger/1.0.0" {
			t.Errorf("User-Agent = %q, want perfmonger/1.0.0", ua)
		}
	}
}

func TestSystemUpdateYesSkipsPrompt(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.1.0", 0o755)
	u := newTestUpdater(srv, exe, "1.1.0")
	u.AssumeYes = true // stdin is not a terminal

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.2.0"))
	assertFileMode(t, exe, 0o755)
	assertNoTempFiles(t, dir)
	if strings.Contains(u.stdout.String(), "[y/N]") {
		t.Errorf("--yes must not prompt:\n%s", u.stdout)
	}
}

func TestSystemUpdateReplacesSymlinkTarget(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	realDir := filepath.Join(t.TempDir(), "real")
	linkDir := filepath.Join(t.TempDir(), "bin")
	for _, d := range []string{realDir, linkDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	realExe := installFakeBinary(t, realDir, "1.0.0", 0o700)
	link := filepath.Join(linkDir, "perfmonger")
	if err := os.Symlink(realExe, link); err != nil {
		t.Fatal(err)
	}
	u := newTestUpdater(srv, link, "1.0.0")
	u.AssumeYes = true

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink itself was replaced; its target should have been")
	}
	assertFileContent(t, realExe, fakeBinaryScript("1.2.0"))
	assertFileMode(t, realExe, 0o700)
	assertNoTempFiles(t, realDir)
	assertNoTempFiles(t, linkDir)
}

func TestSystemUpdateUpToDate(t *testing.T) {
	for _, current := range []string{"1.2.0", "1.3.0"} {
		t.Run(current, func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
			dir := t.TempDir()
			exe := installFakeBinary(t, dir, current, 0o755)
			u := newTestUpdater(srv, exe, current)
			u.AssumeYes = true

			if err := u.run(context.Background()); err != nil {
				t.Fatalf("run() error: %v", err)
			}
			if !strings.Contains(u.stdout.String(), "perfmonger is up to date.") {
				t.Errorf("stdout:\n%s", u.stdout)
			}
			if n := srv.downloadCount(); n != 0 {
				t.Errorf("up-to-date run made %d download requests", n)
			}
			assertFileContent(t, exe, fakeBinaryScript(current))
			assertNoTempFiles(t, dir)
		})
	}
}

func TestSystemUpdateCheckOnly(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{current: "1.0.0", want: "A newer version is available."},
		{current: "1.2.0", want: "perfmonger is up to date."},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
			dir := t.TempDir()
			exe := installFakeBinary(t, dir, tt.current, 0o755)
			u := newTestUpdater(srv, exe, tt.current)
			u.CheckOnly = true
			u.answerPrompt("y\n")

			if err := u.run(context.Background()); err != nil {
				t.Fatalf("run() error: %v", err)
			}
			out := u.stdout.String()
			for _, want := range []string{"Current version: " + tt.current, "Latest version:  1.2.0", tt.want} {
				if !strings.Contains(out, want) {
					t.Errorf("stdout does not contain %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "[y/N]") {
				t.Errorf("--check must not prompt:\n%s", out)
			}
			if n := srv.downloadCount(); n != 0 {
				t.Errorf("--check made %d download requests", n)
			}
			assertFileContent(t, exe, fakeBinaryScript(tt.current))
			assertNoTempFiles(t, dir)
		})
	}
}

// TestSystemUpdateFailuresLeaveBinaryUntouched runs the updater against
// broken releases and checks that every failure keeps the installed binary
// and leaves no temp file behind.
func TestSystemUpdateFailuresLeaveBinaryUntouched(t *testing.T) {
	tests := []struct {
		name    string
		assets  func(t *testing.T) map[string][]byte
		limits  func(*updateLimits)
		wantErr string
	}{
		{
			name: "checksum mismatch",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				a[checksumsAssetName] = []byte(checksumLine([]byte("something else"), testAssetName))
				return a
			},
			wantErr: "SHA256 mismatch",
		},
		{
			name: "tarball tampered after checksums were published",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				a[testAssetName] = buildTarball(t, standardEntries("6.6.6"))
				return a
			},
			wantErr: "SHA256 mismatch",
		},
		{
			name: "missing checksum entry",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				a[checksumsAssetName] = []byte(checksumLine(a[testAssetName], "perfmonger_linux_arm64.tar.gz"))
				return a
			},
			wantErr: "no checksum for " + testAssetName,
		},
		{
			name: "malformed checksums.txt",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				a[checksumsAssetName] = []byte("<html>not a checksum file</html>\n")
				return a
			},
			wantErr: "malformed line 1",
		},
		{
			name: "checksums.txt not found",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				delete(a, checksumsAssetName)
				return a
			},
			wantErr: "404",
		},
		{
			name: "tarball not found",
			assets: func(t *testing.T) map[string][]byte {
				a := releaseAssets(t, standardEntries("1.2.0"))
				delete(a, testAssetName)
				return a
			},
			wantErr: "404",
		},
		{
			name: "tarball larger than the limit",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, standardEntries("1.2.0"))
			},
			limits:  func(l *updateLimits) { l.TarballSize = 16 },
			wantErr: "exceeds the limit",
		},
		{
			name: "checksums.txt larger than the limit",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, standardEntries("1.2.0"))
			},
			limits:  func(l *updateLimits) { l.ChecksumsSize = 16 },
			wantErr: "exceeds the limit",
		},
		{
			name: "binary larger than the limit",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, standardEntries("1.2.0"))
			},
			limits:  func(l *updateLimits) { l.BinarySize = 8 },
			wantErr: "too large",
		},
		{
			name: "smoke test reports the wrong version",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, standardEntries("9.9.9"))
			},
			wantErr: "reports version 9.9.9, expected 1.2.0",
		},
		{
			name: "smoke test fails to run",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "perfmonger", Body: []byte("#!/bin/sh\nexit 3\n")}})
			},
			wantErr: "failed to run",
		},
		{
			name: "smoke test prints garbage",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "perfmonger", Body: []byte("#!/bin/sh\necho hello\n")}})
			},
			wantErr: "unexpected version string",
		},
		{
			name: "archive without perfmonger",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "NEWS.md", Body: []byte(testNews)}})
			},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name: "archive with a path-traversal perfmonger",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "../perfmonger", Body: fakeBinaryScript("1.2.0")}})
			},
			wantErr: "does not contain a perfmonger binary",
		},
		{
			name: "archive with a symlink named perfmonger",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "perfmonger", Typeflag: tar.TypeSymlink, Linkname: "/bin/sh"}})
			},
			wantErr: "not a regular file",
		},
		{
			name: "archive with a directory named perfmonger",
			assets: func(t *testing.T) map[string][]byte {
				return releaseAssets(t, []tarEntry{{Name: "perfmonger", Typeflag: tar.TypeDir}})
			},
			wantErr: "not a regular file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", tt.assets(t))
			root := t.TempDir()
			dir := filepath.Join(root, "bin")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			exe := installFakeBinary(t, dir, "1.0.0", 0o755)
			u := newTestUpdater(srv, exe, "1.0.0")
			u.AssumeYes = true
			if tt.limits != nil {
				tt.limits(&u.Limits)
			}

			err := u.run(context.Background())
			assertErrorContains(t, err, tt.wantErr)
			assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
			assertFileMode(t, exe, 0o755)
			assertNoTempFiles(t, dir)
			if names := dirEntryNames(t, dir); len(names) != 1 || names[0] != "perfmonger" {
				t.Errorf("entries of %s = %v, want [perfmonger]", dir, names)
			}
			if names := dirEntryNames(t, root); len(names) != 1 || names[0] != "bin" {
				t.Errorf("entries of %s = %v, want [bin]", root, names)
			}
		})
	}
}

// TestSystemUpdateIgnoresOtherArchiveEntries checks that only the perfmonger
// binary reaches the disk, even when the archive carries path-traversal or
// nested entries next to it.
func TestSystemUpdateIgnoresOtherArchiveEntries(t *testing.T) {
	entries := append(standardEntries("1.2.0"),
		tarEntry{Name: "../escaped", Body: []byte("escaped\n")},
		tarEntry{Name: "../../escaped-twice", Body: []byte("escaped\n")},
		tarEntry{Name: "sub/../../escaped-nested", Body: []byte("escaped\n")},
		tarEntry{Name: "../perfmonger", Body: fakeBinaryScript("6.6.6")},
		tarEntry{Name: "evil-link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
	)
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, entries))
	root := t.TempDir()
	dir := filepath.Join(root, "a", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.AssumeYes = true

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.2.0"))
	if names := dirEntryNames(t, dir); len(names) != 1 || names[0] != "perfmonger" {
		t.Errorf("entries of %s = %v, want [perfmonger]", dir, names)
	}
	if names := dirEntryNames(t, filepath.Join(root, "a")); len(names) != 1 || names[0] != "bin" {
		t.Errorf("entries of %s = %v, want [bin]", filepath.Join(root, "a"), names)
	}
	if names := dirEntryNames(t, root); len(names) != 1 || names[0] != "a" {
		t.Errorf("entries of %s = %v, want [a]", root, names)
	}
}

func TestSystemUpdateMissingNewsWarns(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, []tarEntry{{Name: "perfmonger", Body: fakeBinaryScript("1.2.0")}}))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.AssumeYes = true

	if err := u.run(context.Background()); err != nil {
		t.Fatalf("run() error: %v", err)
	}
	if !strings.Contains(u.stderr.String(), "NEWS.md is missing") {
		t.Errorf("stderr does not warn about NEWS.md: %q", u.stderr)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.2.0"))
}

func TestSystemUpdateUnwritableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	u := newTestUpdater(srv, exe, "1.0.0")
	u.answerPrompt("y\n")

	err := u.run(context.Background())
	assertErrorContains(t, err, "cannot update "+exe, "sudo")
	if strings.Contains(u.stdout.String(), "[y/N]") {
		t.Errorf("the writability check must fail before prompting:\n%s", u.stdout)
	}
	if n := srv.downloadCount(); n != 0 {
		t.Errorf("made %d download requests despite the unwritable directory", n)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
	assertNoTempFiles(t, dir)
}

func TestSystemUpdateRefusedBeforeNetworkAccess(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		goarch  string
		version string
		wantErr string
	}{
		{name: "dev build", goos: "linux", goarch: "amd64", version: "dev", wantErr: `"dev"`},
		{name: "pre-release version", goos: "linux", goarch: "amd64", version: "1.2.0-rc1", wantErr: "not a release version"},
		{name: "empty version", goos: "linux", goarch: "amd64", version: "", wantErr: "not a release version"},
		{name: "darwin", goos: "darwin", goarch: "arm64", version: "1.0.0", wantErr: "not supported on darwin/arm64"},
		{name: "unsupported arch", goos: "linux", goarch: "386", version: "1.0.0", wantErr: "not supported on linux/386"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
			dir := t.TempDir()
			exe := installFakeBinary(t, dir, "1.0.0", 0o755)
			u := newTestUpdater(srv, exe, tt.version)
			u.GOOS, u.GOARCH = tt.goos, tt.goarch
			u.AssumeYes = true

			err := u.run(context.Background())
			assertErrorContains(t, err, tt.wantErr)
			if n := srv.requestCount("/"); n != 0 {
				t.Errorf("made %d requests; the refusal must happen before any network access", n)
			}
			assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
			assertNoTempFiles(t, dir)
		})
	}
}

func TestSystemUpdateInvalidLatestTagDoesNotDownload(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	srv.latestLocation = "/releases/tag/v1.2.0%2F..%2F..%2Fx"
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.AssumeYes = true

	err := u.run(context.Background())
	assertErrorContains(t, err, "invalid release tag")
	if n := srv.downloadCount(); n != 0 {
		t.Errorf("made %d download requests after an invalid tag", n)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
	assertNoTempFiles(t, dir)
}

func TestSystemUpdateNonTTYWithoutYes(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.Stdin = strings.NewReader("y\n") // piped input must not count as confirmation

	err := u.run(context.Background())
	assertErrorContains(t, err, "not a terminal", "--yes")
	if n := srv.tarballCount(); n != 0 {
		t.Errorf("downloaded the tarball %d times before failing", n)
	}
	assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
	assertNoTempFiles(t, dir)
}

func TestSystemUpdatePromptDeclined(t *testing.T) {
	for _, answer := range []string{"n\n", "N\n", "\n", "", "maybe\n", "no"} {
		t.Run(fmt.Sprintf("%q", answer), func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
			dir := t.TempDir()
			exe := installFakeBinary(t, dir, "1.0.0", 0o755)
			u := newTestUpdater(srv, exe, "1.0.0")
			u.answerPrompt(answer)

			if err := u.run(context.Background()); err != nil {
				t.Fatalf("run() error: %v", err)
			}
			out := u.stdout.String()
			if !strings.Contains(out, "[y/N]") || !strings.Contains(out, "Update cancelled.") {
				t.Errorf("stdout:\n%s", out)
			}
			assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
			assertNoTempFiles(t, dir)
		})
	}
}

func TestSystemUpdatePromptAccepted(t *testing.T) {
	for _, answer := range []string{"y\n", "Y\n", "yes\n", " YES \n", "y"} {
		t.Run(fmt.Sprintf("%q", answer), func(t *testing.T) {
			srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
			dir := t.TempDir()
			exe := installFakeBinary(t, dir, "1.0.0", 0o755)
			u := newTestUpdater(srv, exe, "1.0.0")
			u.answerPrompt(answer)

			if err := u.run(context.Background()); err != nil {
				t.Fatalf("run() error: %v", err)
			}
			assertFileContent(t, exe, fakeBinaryScript("1.2.0"))
			assertNoTempFiles(t, dir)
		})
	}
}

// TestSystemUpdateCancelledDuringPrompt simulates SIGINT while the prompt waits
// for input: the run must return promptly and clean up the temp file.
func TestSystemUpdateCancelledDuringPrompt(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	stdinReader, stdinWriter := io.Pipe()
	t.Cleanup(func() { _ = stdinWriter.Close() })
	u := newTestUpdater(srv, exe, "1.0.0")
	u.Stdin = stdinReader
	u.IsTerminal = func() bool { return true }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- u.run(ctx) }()

	// Wait until the prompt has been printed, then cancel.
	deadline := time.Now().Add(10 * time.Second)
	for srv.tarballCount() == 0 || len(dirEntryNames(t, dir)) != 2 {
		if time.Now().After(deadline) {
			t.Fatal("the updater did not reach the prompt")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("run() returned nil after cancellation")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after cancellation")
	}
	assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
	assertNoTempFiles(t, dir)
}

func TestSystemUpdateCancelledContext(t *testing.T) {
	srv := newFakeReleaseServer(t, "v1.2.0", releaseAssets(t, standardEntries("1.2.0")))
	dir := t.TempDir()
	exe := installFakeBinary(t, dir, "1.0.0", 0o755)
	u := newTestUpdater(srv, exe, "1.0.0")
	u.AssumeYes = true

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := u.run(ctx); err == nil {
		t.Fatal("run() with a cancelled context returned nil")
	}
	assertFileContent(t, exe, fakeBinaryScript("1.0.0"))
	assertNoTempFiles(t, dir)
}

// ---------------------------------------------------------------------------
// Production wiring
// ---------------------------------------------------------------------------

func TestNewUpdateHTTPClient(t *testing.T) {
	client := newUpdateHTTPClient()
	if client.Timeout <= 0 {
		t.Error("the HTTP client must have an overall timeout")
	}
	mustParse := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	via := []*http.Request{{URL: mustParse("https://github.com/hayamiz/perfmonger/releases/download/v1.2.0/checksums.txt")}}
	if err := client.CheckRedirect(&http.Request{URL: mustParse("https://objects.githubusercontent.com/x")}, via); err != nil {
		t.Errorf("https -> https redirect refused: %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: mustParse("http://objects.githubusercontent.com/x")}, via); err == nil {
		t.Error("https -> http redirect must be refused")
	}
	many := make([]*http.Request, 10)
	for i := range many {
		many[i] = via[0]
	}
	if err := client.CheckRedirect(&http.Request{URL: mustParse("https://github.com/x")}, many); err == nil {
		t.Error("more than 10 redirects must be refused")
	}
}

func TestIsTerminalReader(t *testing.T) {
	if isTerminalReader(strings.NewReader("y\n")) {
		t.Error("a strings.Reader is not a terminal")
	}
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminalReader(f) {
		t.Error("a regular file is not a terminal")
	}
}

func TestNewSystemUpdateCommand(t *testing.T) {
	cmd := newSystemUpdateCommand()
	if cmd.Use != "system-update" {
		t.Errorf("Use = %q, want %q", cmd.Use, "system-update")
	}

	expectedFlags := []string{"check", "yes"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}
	if f := cmd.Flags().ShorthandLookup("y"); f == nil || f.Name != "yes" {
		t.Error("expected short flag -y for --yes")
	}

	found := false
	for _, sub := range newRootCommand().Commands() {
		if sub.Name() == "system-update" {
			found = true
		}
	}
	if !found {
		t.Error("system-update is not registered on the root command")
	}
}

// TestSystemUpdateCommandRefusesDevBuild runs the real command wiring. Test
// binaries are built without -ldflags, so VERSION is "dev" and the command must
// refuse before touching the network or the test binary.
func TestSystemUpdateCommandRefusesDevBuild(t *testing.T) {
	if VERSION != "dev" {
		t.Skipf("VERSION is %q, not dev", VERSION)
	}
	root := newRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"system-update", "--yes"})

	err := root.Execute()
	assertErrorContains(t, err, `"dev"`, "install a release build manually")
	if strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("a runtime refusal must not print usage:\n%s", stderr.String())
	}
}

func TestSystemUpdateCommandRejectsArgs(t *testing.T) {
	root := newRootCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"system-update", "v1.2.0"})
	if err := root.Execute(); err == nil {
		t.Error("system-update must reject positional arguments")
	}
}
