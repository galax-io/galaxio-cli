package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunFindsAvailableUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/galax-io/galaxio-cli/releases/latest" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Fatalf("expected GitHub JSON accept header, got %q", got)
		}
		fmt.Fprintf(w, `{
			"tag_name":"v1.2.3",
			"assets":[
				{"name":"galaxio_1.2.3_linux_amd64.tar.gz","browser_download_url":"%s/archive"}
			]
		}`, serverURL(r))
	}))
	defer server.Close()

	result, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.2",
		GOOS:           "linux",
		GOARCH:         "amd64",
		DryRun:         true,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Updated {
		t.Fatal("expected dry run not to update")
	}
	if result.TargetVersion != "1.2.3" {
		t.Fatalf("expected target version 1.2.3, got %q", result.TargetVersion)
	}
	if result.AssetName != "galaxio_1.2.3_linux_amd64.tar.gz" {
		t.Fatalf("expected selected asset name, got %q", result.AssetName)
	}
}

func TestRunSkipsCurrentVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[]}`))
	}))
	defer server.Close()

	result, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.3",
		DryRun:         true,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Updated {
		t.Fatal("expected no update")
	}
	if result.AssetName != "" {
		t.Fatalf("expected no asset selection, got %q", result.AssetName)
	}
}

func TestRunSkipsOlderRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[]}`))
	}))
	defer server.Close()

	result, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.4-0.20260429122805-b25dd5b9f28b+dirty",
		DryRun:         true,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Updated {
		t.Fatal("expected no update")
	}
	if result.AssetName != "" {
		t.Fatalf("expected no asset selection, got %q", result.AssetName)
	}
}

func TestRunInstallsVerifiedArchive(t *testing.T) {
	const assetName = "galaxio_1.2.3_linux_amd64.tar.gz"
	archivePayload := tarGzipArchive(t, "galaxio", "new binary")
	checksum := sha256.Sum256(archivePayload)
	checksums := hex.EncodeToString(checksum[:]) + "  " + assetName + "\n"

	executable := filepath.Join(t.TempDir(), "galaxio")
	if err := os.WriteFile(executable, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/galax-io/galaxio-cli/releases/latest":
			fmt.Fprintf(w, `{
				"tag_name":"v1.2.3",
				"assets":[
					{"name":"%s","browser_download_url":"%s/archive"},
					{"name":"checksums.txt","browser_download_url":"%s/checksums.txt"}
				]
			}`, assetName, serverURL(r), serverURL(r))
		case "/archive":
			_, _ = w.Write(archivePayload)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksums))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	result, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.2",
		Executable:     executable,
		GOOS:           "linux",
		GOARCH:         "amd64",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Updated {
		t.Fatal("expected update to be installed")
	}

	got, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if string(got) != "new binary" {
		t.Fatalf("expected updated binary, got %q", got)
	}
}

func TestRunRejectsChecksumMismatch(t *testing.T) {
	const assetName = "galaxio_1.2.3_linux_amd64.tar.gz"
	archivePayload := tarGzipArchive(t, "galaxio", "new binary")
	checksums := strings.Repeat("0", 64) + "  " + assetName + "\n"
	executable := filepath.Join(t.TempDir(), "galaxio")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/galax-io/galaxio-cli/releases/latest":
			fmt.Fprintf(w, `{
				"tag_name":"v1.2.3",
				"assets":[
					{"name":"%s","browser_download_url":"%s/archive"},
					{"name":"checksums.txt","browser_download_url":"%s/checksums.txt"}
				]
			}`, assetName, serverURL(r), serverURL(r))
		case "/archive":
			_, _ = w.Write(archivePayload)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksums))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.2",
		Executable:     executable,
		GOOS:           "linux",
		GOARCH:         "amd64",
	})

	if err == nil {
		t.Fatal("expected checksum error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestRunUsesRequestedVersionEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/galax-io/galaxio-cli/releases/tags/v1.2.3" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"tag_name":"v1.2.3",
			"assets":[{"name":"galaxio_1.2.3_darwin_arm64.tar.gz","browser_download_url":"https://example.invalid/archive"}]
		}`))
	}))
	defer server.Close()

	_, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		TargetVersion:  "1.2.3",
		CurrentVersion: "1.2.2",
		GOOS:           "darwin",
		GOARCH:         "arm64",
		DryRun:         true,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{name: "newer patch", current: "1.2.3", target: "1.2.4", want: true},
		{name: "same version", current: "1.2.3", target: "v1.2.3", want: false},
		{name: "older target", current: "1.2.4", target: "1.2.3", want: false},
		{name: "build metadata ignored", current: "1.2.3+dirty", target: "1.2.3", want: false},
		{name: "pseudo version core", current: "1.2.4-0.20260429122805-b25dd5b9f28b+dirty", target: "1.2.3", want: false},
		{name: "unknown current", current: "dev", target: "1.2.3", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewerVersion(tt.current, tt.target); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRunExplainsReleaseNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, err := Updater{HTTPClient: server.Client()}.Run(context.Background(), Options{
		APIBase:        server.URL,
		CurrentVersion: "1.2.2",
		DryRun:         true,
	})

	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "release was not found or repository is not accessible") {
		t.Fatalf("expected actionable not found error, got %v", err)
	}
}

func TestExtractBinaryFromZip(t *testing.T) {
	payload := zipArchive(t, "galaxio.exe", "windows binary")

	got, err := extractBinary("galaxio_1.2.3_windows_amd64.zip", payload, "galaxio.exe")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(got) != "windows binary" {
		t.Fatalf("expected extracted payload, got %q", got)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}

func tarGzipArchive(t *testing.T, name string, body string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	payload := []byte(body)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(payload)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tarWriter.Write(payload); err != nil {
		t.Fatalf("write tar payload: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return buffer.Bytes()
}

func zipArchive(t *testing.T, name string, body string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	if _, err := file.Write([]byte(body)); err != nil {
		t.Fatalf("write zip payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buffer.Bytes()
}
