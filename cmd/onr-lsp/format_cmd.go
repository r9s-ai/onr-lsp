package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/r9s-ai/onr-lsp/internal/lsp"
)

func maybeRunFormat(args []string, in io.Reader, out io.Writer, errOut io.Writer) (bool, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) != "format" {
		return false, nil
	}

	fs := flag.NewFlagSet("format", flag.ContinueOnError)
	fs.SetOutput(errOut)
	tabSize := fs.Int("tab-size", 2, "tab size when using spaces")
	useTabs := fs.Bool("tabs", false, "use tabs for indentation")
	write := fs.Bool("write", false, "write result back to file")
	fs.BoolVar(write, "w", false, "write result back to file")

	if err := fs.Parse(args[1:]); err != nil {
		return true, err
	}
	rest := fs.Args()
	if len(rest) > 1 {
		return true, errors.New("format accepts at most one file path")
	}

	path := "-"
	if len(rest) == 1 {
		path = strings.TrimSpace(rest[0])
		if path == "" {
			path = "-"
		}
	}

	src, err := readFormatSource(path, in)
	if err != nil {
		return true, err
	}
	formatted := lsp.FormatText(string(src), lsp.FormatOptions{
		TabSize:      *tabSize,
		InsertSpaces: !*useTabs,
	})

	if *write {
		return true, writeFormattedOutput(path, src, formatted)
	}
	_, err = io.WriteString(out, formatted)
	return true, err
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
