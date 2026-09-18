package main

import "syscall"

// ioctlReadTermios is the request that reads a terminal's attributes on Linux.
const ioctlReadTermios = syscall.TCGETS
