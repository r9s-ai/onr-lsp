package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newVersionCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(
				opts.Stdout,
				"onr-lsp version=%s commit=%s build_date=%s\n",
				opts.BuildInfo.Version,
				opts.BuildInfo.Commit,
				strings.TrimSpace(opts.BuildInfo.BuildDate),
			)
			return err
		},
	}
}
