package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

type Options struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	BuildInfo   BuildInfo
	ServeRunner ServeRunner
}

func Run(args []string, opts Options) error {
	resolved := normalizeOptions(opts)
	root := newRootCmd(resolved)
	root.SetArgs(args)
	return root.Execute()
}

func normalizeOptions(opts Options) Options {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.ServeRunner == nil {
		opts.ServeRunner = defaultServeRunner
	}
	return opts
}

func newRootCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "onr-lsp",
		Short:         "ONR LSP server CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServeWithOptions(opts)
		},
	}
	cmd.SetIn(opts.Stdin)
	cmd.SetOut(opts.Stdout)
	cmd.SetErr(opts.Stderr)
	cmd.AddCommand(
		newServeCmd(opts),
		newFormatCmd(opts),
		newVersionCmd(opts),
	)
	return cmd
}
