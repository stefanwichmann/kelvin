package main

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Masterminds/semver"
)

// matchingAssetJSON returns a release asset entry that matches the test
// platform, so release-info parsing finds a candidate.
func matchingAssetJSON() string {
	return fmt.Sprintf(`{"name":"kelvin_1.0.0_%s_%s.tar.gz","content_type":"application/gzip","browser_download_url":"http://example.invalid/kelvin.tar.gz"}`, runtime.GOOS, runtime.GOARCH)
}

// TestExtractBinaryFromCorruptTarArchive pins crash safety: a corrupt
// archive must produce an error, never terminate the daemon (issue #131).
func TestExtractBinaryFromCorruptTarArchive(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatalf("could not create archive: %v", err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte("this is not a tar archive")); err != nil {
		t.Fatalf("could not write archive: %v", err)
	}
	writer.Close()
	file.Close()

	if _, err := extractBinaryFromTarArchive(archive, "kelvin", t.TempDir()); err == nil {
		t.Error("corrupt tar archive must return an error")
	}
}

// TestDownloadLatestReleaseInfo pins metadata robustness: unexpected JSON
// shapes and error responses produce errors, never panics (issue #131).
func TestDownloadLatestReleaseInfo(t *testing.T) {
	valid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v1.4.0","assets":[%s]}`, matchingAssetJSON())
	}))
	defer valid.Close()
	name, url, err := downloadLatestReleaseInfo(valid.URL)
	if err != nil || name != "v1.4.0" || url == "" {
		t.Errorf("valid release info: got (%q, %q, %v)", name, url, err)
	}

	malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":42,"assets":"nope"}`)
	}))
	defer malformed.Close()
	if _, _, err := downloadLatestReleaseInfo(malformed.URL); err == nil {
		t.Error("malformed release info must return an error")
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer failing.Close()
	if _, _, err := downloadLatestReleaseInfo(failing.URL); err == nil {
		t.Error("non-200 release info response must return an error")
	}
}

// TestDownloadReleaseArchiveChecksStatus pins that an error response is
// detected instead of saving the error page as the archive (issue #131).
func TestDownloadReleaseArchiveChecksStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := downloadReleaseArchive(server.URL); err == nil {
		t.Error("non-200 archive response must return an error")
	}
}

// TestUpdateAvailableRefusesMajorUpgrade pins the guard that holds the
// installed fleet at v1.x when v2.0.0 releases: a major version jump is
// refused unless forced.
func TestUpdateAvailableRefusesMajorUpgrade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","assets":[%s]}`, matchingAssetJSON())
	}))
	defer server.Close()

	current := semver.MustParse("1.3.9")
	avail, _, err := updateAvailable(current, server.URL, false)
	if err != nil {
		t.Fatalf("updateAvailable failed: %v", err)
	}
	if avail {
		t.Error("major upgrade must be refused without forceUpdate")
	}
	forced, _, err := updateAvailable(current, server.URL, true)
	if err != nil || !forced {
		t.Errorf("forced major upgrade must be offered: got (%v, %v)", forced, err)
	}
}
