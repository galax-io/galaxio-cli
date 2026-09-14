package reporttest

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"
)

func corpusLog(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "testdata", "corpus", "gatling", "3.12.0", "simulation.log"))
	if err != nil {
		t.Fatalf("read corpus log: %v", err)
	}
	return data
}

func TestSplit(t *testing.T) {
	t.Parallel()

	log := corpusLog(t)
	header, body, err := Split(log)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if !bytes.HasSuffix(header, []byte("\n")) || !bytes.Contains(header, []byte("\nRUN\t")) {
		t.Errorf("header does not end with the RUN line: %q", header)
	}
	if bytes.Count(header, []byte("\nRUN\t")) != 1 || bytes.Contains(body, []byte("RUN\t")) {
		t.Errorf("the RUN line is not the boundary")
	}
	if !bytes.Equal(append(append([]byte(nil), header...), body...), log) {
		t.Errorf("header + body is not the log")
	}

	if _, _, err := Split([]byte("ASSERTION\tx\nUSER\ty\n")); err == nil {
		t.Errorf("Split of a log without a RUN record succeeded")
	}
}

func TestReplay(t *testing.T) {
	t.Parallel()

	header, body, err := Split(corpusLog(t))
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	tests := []struct {
		name string
		n    int
	}{
		{name: "header only", n: 0},
		{name: "the log itself", n: 1},
		{name: "three bodies", n: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expected := append([]byte(nil), header...)
			for range tt.n {
				expected = append(expected, body...)
			}
			if err := iotest.TestReader(Replay(header, body, tt.n), expected); err != nil {
				t.Fatalf("Replay does not behave as an io.Reader: %v", err)
			}
			got, err := io.ReadAll(Replay(header, body, tt.n))
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("Replay yielded %d bytes, want %d", len(got), len(expected))
			}
		})
	}
}
