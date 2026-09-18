// Package reporttest is what the tests and benchmarks of internal/report share:
// synthetic Gatling logs, the comparison of a summary with what Gatling
// recorded, and the service the live Gatling runs load.
//
// A recorded log is a few kilobytes; the memory goal is about logs of
// gigabytes. Replay yields a text log of any size from one recording without
// writing a file or holding the log in memory, which is what lets a test
// measure the reader rather than the fixture.
package reporttest

import (
	"bytes"
	"errors"
	"io"
)

// runPrefix opens the RUN record of a text simulation.log, the last line of
// its header.
var runPrefix = []byte("RUN\t")

// Split divides a text simulation.log into its header — every line through
// the RUN record — and its body, the events after it. Both keep their trailing
// newline, so header followed by body is the log again.
//
// The header is capped to its own length, so appending to it cannot reach into
// the body it was cut from. A log whose last line is unterminated is refused:
// its body would end mid-record, and repeating it would splice that record
// onto the next copy's first one.
func Split(log []byte) (header, body []byte, err error) {
	offset := 0
	for offset < len(log) {
		end := bytes.IndexByte(log[offset:], '\n')
		if end < 0 {
			break
		}
		lineEnd := offset + end + 1
		if bytes.HasPrefix(log[offset:], runPrefix) {
			body = log[lineEnd:]
			if len(body) > 0 && body[len(body)-1] != '\n' {
				return nil, nil, errors.New("reporttest: the log's last line is unterminated")
			}

			return log[:lineEnd:lineEnd], body, nil
		}
		offset = lineEnd
	}
	return nil, nil, errors.New("reporttest: no RUN record in the log")
}

// Replay returns a reader that yields header once and then body n times.
// Reads are served from the recording's bytes directly; nothing is copied
// ahead of time, so a replay of gigabytes costs the size of one recording.
//
// A negative n yields the header alone, as zero does.
func Replay(header, body []byte, n int) io.Reader {
	return &replay{body: body, remaining: max(n, 0), segment: header}
}

type replay struct {
	body      []byte
	remaining int    // body repetitions still to serve after the current segment
	segment   []byte // the bytes of the current segment not yet read
}

func (r *replay) Read(p []byte) (int, error) {
	for len(r.segment) == 0 {
		if r.remaining <= 0 {
			return 0, io.EOF
		}
		r.remaining--
		r.segment = r.body
	}
	n := copy(p, r.segment)
	r.segment = r.segment[n:]
	return n, nil
}
