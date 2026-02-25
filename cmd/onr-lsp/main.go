package main

import (
	"log"
	"os"

	"github.com/r9s-ai/onr-lsp/internal/lsp"
)

func main() {
	logger := log.New(os.Stderr, "onr-lsp: ", log.LstdFlags|log.Lshortfile)
	srv := lsp.NewServer(os.Stdin, os.Stdout, logger)
	if err := srv.Run(); err != nil {
		logger.Fatalf("server exited with error: %v", err)
	}
}
