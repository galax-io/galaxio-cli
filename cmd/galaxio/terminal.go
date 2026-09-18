package main

import (
	"io"
	"os"
)

// ansiTerminal reports whether w is a terminal that understands escape
// sequences, which is what the colours of the summary and the progress block
// need. A pipe, a file and a device that is no terminal — /dev/null among them
// — are not.
//
// Whether there is a terminal is asked of the device itself, because a
// character device is not a terminal by being one: writing to /dev/null would
// cost the walk a redraw five times a second and put control sequences where
// FR-037 allows none. What the terminal understands is asked of the platform:
// TERM where a terminal sets it, and the console's own mode on Windows, which
// has none.
func ansiTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	return understandsEscapes(f)
}
