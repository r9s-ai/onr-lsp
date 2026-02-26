package main

import (
	"fmt"
	"os"

	"github.com/r9s-ai/onr-lsp/cli"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = ""
)

func main() {
	opts := cli.Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		BuildInfo: cli.BuildInfo{
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
		},
	}
	if err := cli.Run(os.Args[1:], opts); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
