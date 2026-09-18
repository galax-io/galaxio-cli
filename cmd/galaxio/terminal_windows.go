package main

import (
	"os"
	"syscall"
)

// enableVirtualTerminalProcessing is the console mode flag under which a
// Windows console interprets escape sequences rather than prints them.
const enableVirtualTerminalProcessing = 0x0004

// understandsEscapes reports whether f is a console that interprets escape
// sequences. Windows sets no TERM, and interpreting them is a mode the console
// holds: without it a console prints the sequences as text and scrolls where
// the block means to redraw. The mode is read and never changed, so a command
// that is killed leaves the console as it found it.
func understandsEscapes(f *os.File) bool {
	var mode uint32

	if err := syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode); err != nil {
		return false
	}

	if mode&enableVirtualTerminalProcessing == 0 {
		return false
	}

	// A shell that exports TERM=dumb means it on any platform.
	return os.Getenv("TERM") != "dumb"
}
