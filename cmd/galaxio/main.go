package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// An interrupt cancels the context the commands run under rather than
	// killing the process outright, so that work which polls it — reading a
	// run, which can take minutes on a log large enough — stops where it is
	// and reports why. stop runs before os.Exit, which no defer would.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := executeContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()

	os.Exit(code)
}
