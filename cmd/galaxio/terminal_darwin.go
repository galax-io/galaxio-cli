package main

import "syscall"

// ioctlReadTermios is the request that reads a terminal's attributes on macOS.
const ioctlReadTermios = syscall.TIOCGETA
