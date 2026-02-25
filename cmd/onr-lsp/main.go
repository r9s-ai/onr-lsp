package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/r9s-ai/onr-lsp/internal/lsp"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = ""
)

func main() {
	if maybePrintVersion(os.Args[1:], os.Stdout) {
		return
	}
	lsp.ServerVersion = Version
	logger := log.New(os.Stderr, "onr-lsp: ", log.LstdFlags|log.Lshortfile)
	srv := lsp.NewServer(os.Stdin, os.Stdout, logger)
	if err := srv.Run(); err != nil {
		logger.Fatalf("server exited with error: %v", err)
	}
}

func maybePrintVersion(args []string, w io.Writer) bool {
	if len(args) == 0 {
		return false
	}
	if !isVersionArg(args[0]) {
		return false
	}
	fmt.Fprintf(w, "onr-lsp version=%s commit=%s build_date=%s\n", Version, Commit, strings.TrimSpace(BuildDate))
	return true
}

func isVersionArg(arg string) bool {
	switch strings.TrimSpace(arg) {
	case "--version", "-version", "-v", "version":
		return true
	default:
		return false
	}
}
