//go:build integration

package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/report/reporttest"
	"github.com/galax-io/parsec/gatling/simlog"
	"github.com/galax-io/parsec/model"
)

// tdigestJars are the two releases of com.tdunning:t-digest the etalon runs:
// 3.1, whose AVLTreeDigest Gatling 3.11 and 3.12 use, and 3.3, whose
// MergingDigest is described beside it. Each is pinned by the SHA-256 of the jar
// Maven Central serves, whose published SHA-1 is 451ed219688aed5821a789428fd5e10426d11312
// for 3.1 and 5e96c4fd7d63b05828cf5ef41da20649195b1b78 for 3.3.
var tdigestJars = [2]struct{ version, sha256 string }{
	{"3.1", "271f3a5a4bc79d7554c9e9e557669af83bcbda0db871e0b8c969d56e51c123a9"},
	{"3.3", "dc8be5228e0733e12fbe35eeffb6f720dff576bb2eb5dee69c1c211adcd7aeb9"},
}

// withoutEtalon ends a test that cannot run the etalon: it skips, or fails when
// GALAXIO_ETALON_REQUIRED=1, which CI sets so that a missing JDK or jar is never
// mistaken for the equality holding.
func withoutEtalon(t *testing.T, format string, args ...any) {
	t.Helper()

	if os.Getenv("GALAXIO_ETALON_REQUIRED") == "1" {
		t.Fatalf(format, args...)
	}

	t.Skipf(format, args...)
}

// tdigestJar returns the path of one release's jar, found where a build already
// put it — the directory GALAXIO_TDIGEST_JARS names, the local Maven repository,
// or the Coursier cache sbt fills — and checked against its pinned digest. It
// ends t as withoutEtalon does when the jar is nowhere; a Gatling 3.11 build
// brings 3.1 and a 3.13 build brings 3.3.
func tdigestJar(t *testing.T, version, sum string) string {
	t.Helper()

	name := "t-digest-" + version + ".jar"
	maven := filepath.Join("com", "tdunning", "t-digest", version, name)

	var candidates []string

	if dir := os.Getenv("GALAXIO_TDIGEST_JARS"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, name))
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".m2", "repository", maven))
	}

	if cache, err := os.UserCacheDir(); err == nil {
		for _, coursier := range []string{"Coursier", "coursier"} {
			candidates = append(candidates, filepath.Join(cache, coursier, "v1", "https", "repo1.maven.org", "maven2", maven))
		}
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if got := sha256.Sum256(data); hex.EncodeToString(got[:]) != sum {
			t.Fatalf("%s has SHA-256 %x, not the %s Maven Central serves", path, got, sum)
		}

		return path
	}

	withoutEtalon(t, "no %s in %v; set GALAXIO_TDIGEST_JARS to a directory holding it", name, candidates)

	return ""
}

// runEtalon runs testdata/etalon/Etalon.java over the run rd yields and returns
// what it printed. Without a JDK or the jars it ends t as withoutEtalon does.
func runEtalon(t *testing.T, rd simlog.RunReader) []byte {
	t.Helper()

	if _, err := exec.LookPath("java"); err != nil {
		withoutEtalon(t, "java is not on the path: %v", err)
	}

	args := []string{filepath.Join("testdata", "etalon", "Etalon.java")}
	for _, jar := range tdigestJars {
		args = append(args, tdigestJar(t, jar.version, jar.sha256))
	}

	var samples bytes.Buffer
	if err := reporttest.EtalonSamples(&samples, rd); err != nil {
		t.Fatalf("EtalonSamples: %v", err)
	}

	var stderr bytes.Buffer

	etalon := exec.Command("java", args...)
	etalon.Stdin = &samples
	etalon.Stderr = &stderr

	out, err := etalon.Output()
	if err != nil {
		t.Fatalf("java Etalon.java: %v\n%s", err, stderr.Bytes())
	}

	return out
}

// TestEtalonRecordings reruns the etalon over every recorded run — the five of
// the corpus and the five live ones — and over every synthetic run, and requires
// the etalon.tsv kept beside each byte for byte, so that the ordinary suite
// holds this tool's percentiles to what the real libraries give.
// GALAXIO_ETALON_RECORD=1 writes them instead.
func TestEtalonRecordings(t *testing.T) {
	type recording struct {
		dir string
		run func(t *testing.T) simlog.RunReader
	}

	var recordings []recording

	for _, version := range reporttest.Versions {
		recordings = append(recordings, recording{
			dir: filepath.Join(corpusDir, version),
			run: func(t *testing.T) simlog.RunReader { return openBytes(t, corpusLog(t, version)) },
		})
	}

	for _, version := range liveVersions(t) {
		recordings = append(recordings, recording{
			dir: filepath.Join(liveDir(), version),
			run: func(t *testing.T) simlog.RunReader { return openBytes(t, liveLog(t, version)) },
		})
	}

	for _, synthetic := range syntheticRuns {
		recordings = append(recordings, recording{
			dir: syntheticDir(synthetic.name),
			run: func(*testing.T) simlog.RunReader { return reporttest.Items(model.Run{}, nil, synthetic.items()...) },
		})
	}

	for _, rec := range recordings {
		t.Run(rec.dir, func(t *testing.T) {
			got := runEtalon(t, rec.run(t))
			path := filepath.Join(rec.dir, "etalon.tsv")

			if os.Getenv("GALAXIO_ETALON_RECORD") == "1" {
				if err := os.MkdirAll(rec.dir, 0o755); err != nil {
					t.Fatalf("make %s: %v", rec.dir, err)
				}

				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}

				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			if !bytes.Equal(got, want) {
				t.Errorf("the etalon gives\n%s\nwhere %s holds\n%s", got, path, want)
			}
		})
	}
}
