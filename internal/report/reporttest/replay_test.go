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
	// The boundary is where the header ENDS, which a "contains" test cannot
	// see: the recording opens with ASSERTION lines, so a newline precedes RUN
	// whatever the split does.
	lastLine := header[bytes.LastIndexByte(header[:len(header)-1], '\n')+1:]
	if !bytes.HasPrefix(lastLine, []byte("RUN\t")) {
		t.Errorf("the header's last line is %q, want the RUN record", lastLine)
	}

	if cap(header) != len(header) {
		t.Errorf("cap(header) = %d, len = %d: the header shares its array with the body, so appending to it would overwrite the events", cap(header), len(header))
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

	// A body ending mid-record is what a killed run leaves; repeating it would
	// splice that record onto the next copy's first one.
	if _, _, err := Split([]byte("RUN\ta\tb\nUSER\tunterminated")); err == nil {
		t.Errorf("Split of a log whose last line is unterminated succeeded")
	}

	// A RUN record with nothing after it is a complete log, not an error.
	onlyHeader, emptyBody, err := Split([]byte("RUN\ta\tb\n"))
	if err != nil || len(emptyBody) != 0 || string(onlyHeader) != "RUN\ta\tb\n" {
		t.Errorf("Split of a header-only log = %q, %q, %v", onlyHeader, emptyBody, err)
	}
}

func TestReplayRefusesToRepeatForever(t *testing.T) {
	t.Parallel()

	got, err := io.ReadAll(Replay([]byte("H\n"), []byte("B\n"), -1))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if string(got) != "H\n" {
		t.Fatalf("a negative count yielded %q, want the header alone", got)
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
