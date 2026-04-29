// Package selfupdate updates a running galaxio binary from GitHub Releases.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	defaultAPIBase = "https://api.github.com"
	binaryName     = "galaxio"
)

// Options configures a self-update run.
type Options struct {
	Repo           string
	TargetVersion  string
	CurrentVersion string
	Executable     string
	GOOS           string
	GOARCH         string
	APIBase        string
	DryRun         bool
}

// Result describes the outcome of a self-update run.
type Result struct {
	CurrentVersion string
	TargetVersion  string
	Updated        bool
	DryRun         bool
	AssetName      string
}

// Updater updates galaxio from GitHub Releases.
type Updater struct {
	HTTPClient *http.Client
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Run checks for a release and installs it when an update is available.
func (u Updater) Run(ctx context.Context, opts Options) (Result, error) {
	opts = opts.withDefaults()

	current := cleanVersion(opts.CurrentVersion)
	rel, err := u.fetchRelease(ctx, opts)
	if err != nil {
		return Result{}, err
	}

	target := cleanVersion(rel.TagName)
	result := Result{
		CurrentVersion: current,
		TargetVersion:  target,
		DryRun:         opts.DryRun,
	}

	if !isNewerVersion(current, target) {
		return result, nil
	}

	archiveAsset, err := findAsset(rel.Assets, archiveName(target, opts.GOOS, opts.GOARCH))
	if err != nil {
		return Result{}, err
	}
	result.AssetName = archiveAsset.Name

	if opts.DryRun {
		return result, nil
	}

	checksumAsset, err := findAsset(rel.Assets, "checksums.txt")
	if err != nil {
		return Result{}, err
	}

	archiveBytes, err := u.download(ctx, archiveAsset.BrowserDownloadURL)
	if err != nil {
		return Result{}, fmt.Errorf("download release asset: %w", err)
	}

	checksumBytes, err := u.download(ctx, checksumAsset.BrowserDownloadURL)
	if err != nil {
		return Result{}, fmt.Errorf("download checksums: %w", err)
	}

	if err := verifyChecksum(archiveAsset.Name, archiveBytes, string(checksumBytes)); err != nil {
		return Result{}, err
	}

	binaryBytes, err := extractBinary(archiveAsset.Name, archiveBytes, executableName(opts.GOOS))
	if err != nil {
		return Result{}, err
	}

	if err := installBinary(opts.Executable, binaryBytes); err != nil {
		return Result{}, err
	}

	result.Updated = true
	return result, nil
}

func (opts Options) withDefaults() Options {
	if opts.Repo == "" {
		opts.Repo = "galax-io/galaxio-cli"
	}
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}
	if opts.APIBase == "" {
		opts.APIBase = defaultAPIBase
	}
	return opts
}

func (u Updater) fetchRelease(ctx context.Context, opts Options) (release, error) {
	endpoint := strings.TrimRight(opts.APIBase, "/") + "/repos/" + opts.Repo + "/releases/latest"
	if opts.TargetVersion != "" {
		endpoint = strings.TrimRight(opts.APIBase, "/") + "/repos/" + opts.Repo + "/releases/tags/v" + cleanVersion(opts.TargetVersion)
	}

	payload, err := u.downloadJSON(ctx, endpoint)
	if err != nil {
		return release{}, fmt.Errorf("fetch release metadata: %w", err)
	}

	var rel release
	if err := json.Unmarshal(payload, &rel); err != nil {
		return release{}, fmt.Errorf("decode release metadata: %w", err)
	}
	if rel.TagName == "" {
		return release{}, errors.New("release metadata is missing tag_name")
	}

	return rel, nil
}

func (u Updater) download(ctx context.Context, url string) ([]byte, error) {
	return u.downloadWithHeaders(ctx, url, map[string]string{
		"Accept": "application/octet-stream",
	})
}

func (u Updater) downloadJSON(ctx context.Context, url string) ([]byte, error) {
	return u.downloadWithHeaders(ctx, url, map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
}

func (u Updater) downloadWithHeaders(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	client := u.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "galaxio-cli")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("GET %s returned %s; release was not found or repository is not accessible", url, resp.Status)
		}
		return nil, fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func findAsset(assets []asset, name string) (asset, error) {
	for _, item := range assets {
		if item.Name == name {
			return item, nil
		}
	}
	return asset{}, fmt.Errorf("release asset %q not found", name)
}

func archiveName(version string, goos string, goarch string) string {
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return fmt.Sprintf("galaxio_%s_%s_%s%s", cleanVersion(version), goos, goarch, extension)
}

func executableName(goos string) string {
	if goos == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

func cleanVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func isNewerVersion(current string, target string) bool {
	if current == "" {
		return target != ""
	}

	currentParts, currentOK := versionParts(current)
	targetParts, targetOK := versionParts(target)
	if !currentOK || !targetOK {
		return current != target
	}

	for i := range targetParts {
		if targetParts[i] > currentParts[i] {
			return true
		}
		if targetParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

func versionParts(version string) ([3]int, bool) {
	var parts [3]int
	version = cleanVersion(version)
	if before, _, ok := strings.Cut(version, "-"); ok {
		version = before
	}
	if before, _, ok := strings.Cut(version, "+"); ok {
		version = before
	}

	items := strings.Split(version, ".")
	if len(items) != 3 {
		return parts, false
	}

	for index, item := range items {
		if item == "" {
			return parts, false
		}
		for _, char := range item {
			if char < '0' || char > '9' {
				return parts, false
			}
			parts[index] = parts[index]*10 + int(char-'0')
		}
	}

	return parts, true
}

func verifyChecksum(assetName string, payload []byte, checksums string) error {
	want := ""
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("checksum for %q not found", assetName)
	}

	sum := sha256.Sum256(payload)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %q", assetName)
	}

	return nil
}

func extractBinary(assetName string, payload []byte, executable string) ([]byte, error) {
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractBinaryFromTarGzip(payload, executable)
	case strings.HasSuffix(assetName, ".zip"):
		return extractBinaryFromZip(payload, executable)
	default:
		return nil, fmt.Errorf("unsupported archive format %q", assetName)
	}
}

func extractBinaryFromTarGzip(payload []byte, executable string) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(header.Name) != executable {
			continue
		}
		return io.ReadAll(tarReader)
	}

	return nil, fmt.Errorf("binary %q not found in archive", executable)
}

func extractBinaryFromZip(payload []byte, executable string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		if filepath.Base(file.Name) != executable {
			continue
		}

		item, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer item.Close()

		return io.ReadAll(item)
	}

	return nil, fmt.Errorf("binary %q not found in archive", executable)
}

func installBinary(executable string, payload []byte) error {
	if executable == "" {
		return errors.New("executable path is required")
	}

	dir := filepath.Dir(executable)
	temp, err := os.CreateTemp(dir, ".galaxio-update-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(0o755); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(tempName, executable)
}
