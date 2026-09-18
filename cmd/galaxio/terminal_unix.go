//go:build linux || darwin

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// understandsEscapes reports whether f is a terminal that says it understands
// escape sequences: the kernel gives the device terminal attributes, as
// isatty(3) asks, and TERM names a terminal that is not dumb.
func understandsEscapes(f *os.File) bool {
	var termios syscall.Termios

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, f.Fd(), ioctlReadTermios,
		uintptr(unsafe.Pointer(&termios)), 0, 0, 0); err != 0 {
		return false
	}

	term := os.Getenv("TERM")

	return term != "" && term != "dumb"
}
