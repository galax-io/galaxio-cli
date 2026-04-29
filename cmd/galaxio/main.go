package main

import (
	"os"
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}
