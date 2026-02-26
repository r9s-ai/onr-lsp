package cli

import internalcli "github.com/r9s-ai/onr-lsp/internal/cli"

type BuildInfo = internalcli.BuildInfo
type Options = internalcli.Options

func Run(args []string, opts Options) error {
	return internalcli.Run(args, opts)
}
