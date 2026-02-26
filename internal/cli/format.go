package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/r9s-ai/onr-lsp/internal/lsp"
	"github.com/spf13/cobra"
)

type formatOptions struct {
	tabSize int
	useTabs bool
	write   bool
}

func newFormatCmd(opts Options) *cobra.Command {
	formatOpts := formatOptions{tabSize: 2}
	cmd := &cobra.Command{
		Use:   "format [file|-]",
		Short: "Format ONR DSL document",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("format accepts at most one file path")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "-"
			if len(args) == 1 {
				path = strings.TrimSpace(args[0])
				if path == "" {
					path = "-"
				}
			}

			src, err := readFormatSource(path, opts.Stdin)
			if err != nil {
				return err
			}
			formatted := lsp.FormatText(string(src), lsp.FormatOptions{
				TabSize:      formatOpts.tabSize,
				InsertSpaces: !formatOpts.useTabs,
			})
			if formatOpts.write {
				return writeFormattedOutput(path, src, formatted)
			}
			_, err = io.WriteString(opts.Stdout, formatted)
			return err
		},
	}

	fs := cmd.Flags()
	fs.IntVar(&formatOpts.tabSize, "tab-size", 2, "tab size when using spaces")
	fs.BoolVar(&formatOpts.useTabs, "tabs", false, "use tabs for indentation")
	fs.BoolVarP(&formatOpts.write, "write", "w", false, "write result back to file")
	return cmd
}

func readFormatSource(path string, in io.Reader) ([]byte, error) {
	if path == "-" {
		src, err := io.ReadAll(in)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return src, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return src, nil
}

func writeFormattedOutput(path string, src []byte, formatted string) error {
	if path == "-" {
		return errors.New("--write requires a file path")
	}
	if formatted == string(src) {
		return nil
	}
	mode := os.FileMode(0o644)
	if st, statErr := os.Stat(path); statErr == nil {
		mode = st.Mode().Perm()
	}
	if err := os.WriteFile(path, []byte(formatted), mode); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}
