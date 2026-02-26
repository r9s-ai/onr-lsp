package cli

import (
	"fmt"
	"io"
	"log"

	"github.com/r9s-ai/onr-lsp/internal/lsp"
	"github.com/spf13/cobra"
)

type ServeRuntimeOptions struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	BuildInfo BuildInfo
}

type ServeRunner func(opts ServeRuntimeOptions) error

func newServeCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run ONR LSP server over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServeWithOptions(opts)
		},
	}
}

func runServeWithOptions(opts Options) error {
	return opts.ServeRunner(ServeRuntimeOptions{
		Stdin:     opts.Stdin,
		Stdout:    opts.Stdout,
		Stderr:    opts.Stderr,
		BuildInfo: opts.BuildInfo,
	})
}

func defaultServeRunner(opts ServeRuntimeOptions) error {
	lsp.ServerVersion = opts.BuildInfo.Version
	logger := log.New(opts.Stderr, "onr-lsp: ", log.LstdFlags|log.Lshortfile)
	srv := lsp.NewServer(opts.Stdin, opts.Stdout, logger)
	if err := srv.Run(); err != nil {
		return fmt.Errorf("server exited with error: %w", err)
	}
	return nil
}
