package legacy

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func testDocuments() map[Product][]byte {
	return map[Product][]byte{
		Stats:       []byte("{\"stats\":true}\n"),
		GlobalStats: []byte("{\"global\":true}\n"),
	}
}

func TestPublish(t *testing.T) {
	dir := t.TempDir()
	documents := testDocuments()
	if err := Publish(dir, documents, false); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "js", "stats.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Publish(dir, documents, false); err == nil {
		t.Fatal("existing file was overwritten without --overwrite")
	}
	after, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatal("refused publication changed the existing file")
	}

	documents[Stats] = []byte("{\"stats\":\"updated\"}\n")
	if err := Publish(dir, documents, true); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(updated, documents[Stats]) {
		t.Fatalf("overwritten stats.json = %q, %v", updated, err)
	}
}

func TestPublishWriteFailure(t *testing.T) {
	if dir := os.Getenv("GALAXIO_TEST_LIMITED_PUBLISH_DIR"); dir != "" {
		documents := map[Product][]byte{Stats: []byte(strings.Repeat("x", 4096))}
		err := Publish(dir, documents, os.Getenv("GALAXIO_TEST_OVERWRITE") == "true")
		if err == nil || !strings.Contains(err.Error(), "stats.json") {
			t.Fatalf("Publish error = %v, want write failure naming stats.json", err)
		}
		os.Exit(0) // Do not write the child test runner's coverage report under this limit.
	}
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX file-size limits")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("requires bash for an isolated file-size limit")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name      string
		overwrite string
		existing  bool
	}{
		{name: "create", overwrite: "false"},
		{name: "overwrite absent", overwrite: "true"},
		{name: "overwrite existing", overwrite: "true", existing: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "js", "stats.json")
			if tt.existing {
				if err := Publish(dir, map[Product][]byte{Stats: []byte("original")}, false); err != nil {
					t.Fatal(err)
				}
			}
			// Limit only the child process; a real partial write must not expose a broken output.
			cmd := exec.CommandContext(t.Context(), bash, "-c",
				`trap '' XFSZ; ulimit -f 1 || exit 1; exec "$1" -test.run '^TestPublishWriteFailure$'`, "--", executable)
			cmd.Env = append(os.Environ(), "GALAXIO_TEST_LIMITED_PUBLISH_DIR="+dir, "GALAXIO_TEST_OVERWRITE="+tt.overwrite)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("limited writer: %v\n%s", err, output)
			}
			if tt.existing {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != "original" {
					t.Fatalf("failed overwrite changed original: %d bytes, %v", len(data), err)
				}
			} else if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed write left a partial output: %v", err)
			}
			files, err := os.ReadDir(filepath.Dir(path))
			wantFiles := 0
			if tt.existing {
				wantFiles = 1
			}
			if err != nil || len(files) != wantFiles {
				t.Fatalf("files after failure = %v, %v", files, err)
			}
		})
	}
}

func TestPublishPreflightsAllSelectedFiles(t *testing.T) {
	dir := t.TempDir()
	jsDir := filepath.Join(dir, "js")
	if err := os.Mkdir(jsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte("existing")
	globalPath := filepath.Join(jsDir, "global_stats.json")
	if err := os.WriteFile(globalPath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	err := Publish(dir, testDocuments(), false)
	if err == nil || !strings.Contains(err.Error(), globalPath) {
		t.Fatalf("Publish error = %v, want collision path", err)
	}
	if _, err := os.Stat(filepath.Join(jsDir, "stats.json")); !os.IsNotExist(err) {
		t.Fatalf("preflight created stats.json: %v", err)
	}
	got, err := os.ReadFile(globalPath)
	if err != nil || !reflect.DeepEqual(got, sentinel) {
		t.Fatalf("preflight changed global_stats.json: %q, %v", got, err)
	}
}

func TestPublishRejectsUnsafeDestinations(t *testing.T) {
	t.Run("js regular file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "js")
		if err := os.WriteFile(path, []byte("sentinel"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Publish(dir, testDocuments(), true); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("Publish error = %v", err)
		}
	})

	t.Run("js symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := t.TempDir()
		path := filepath.Join(dir, "js")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if err := Publish(dir, testDocuments(), true); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("Publish error = %v", err)
		}
	})

	t.Run("selected directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "js", "stats.json")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Publish(dir, map[Product][]byte{Stats: testDocuments()[Stats]}, true); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("Publish error = %v", err)
		}
	})

	t.Run("selected symlink", func(t *testing.T) {
		dir := t.TempDir()
		jsDir := filepath.Join(dir, "js")
		if err := os.Mkdir(jsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(dir, "target")
		sentinel := []byte("sentinel")
		if err := os.WriteFile(target, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(jsDir, "stats.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if err := Publish(dir, map[Product][]byte{Stats: testDocuments()[Stats]}, true); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("Publish error = %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil || !reflect.DeepEqual(got, sentinel) {
			t.Fatalf("symlink target changed: %q, %v", got, err)
		}
	})
}
