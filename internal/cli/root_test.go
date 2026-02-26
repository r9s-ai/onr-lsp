package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRootCmdHasSubcommands(t *testing.T) {
	t.Parallel()

	opts := normalizeOptions(Options{
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	root := newRootCmd(opts)

	if _, _, err := root.Find([]string{"serve"}); err != nil {
		t.Fatalf("find serve subcommand: %v", err)
	}
	if _, _, err := root.Find([]string{"format"}); err != nil {
		t.Fatalf("find format subcommand: %v", err)
	}
	if _, _, err := root.Find([]string{"version"}); err != nil {
		t.Fatalf("find version subcommand: %v", err)
	}
}

func TestRunDefaultsToServe(t *testing.T) {
	t.Parallel()

	called := false
	err := Run(nil, Options{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run root command: %v", err)
	}
	if !called {
		t.Fatalf("expected default serve runner to be called")
	}
}

func TestVersionCommandOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Run([]string{"version"}, Options{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		BuildInfo: BuildInfo{
			Version:   "1.2.3",
			Commit:    "abc123",
			BuildDate: "2026-02-26T11:11:11Z\n",
		},
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	if err != nil {
		t.Fatalf("run version command: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "onr-lsp version=1.2.3 commit=abc123 build_date=2026-02-26T11:11:11Z") {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestVersionFlagsRemoved(t *testing.T) {
	t.Parallel()

	opts := Options{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error {
			return errors.New("should not run")
		},
	}

	err := Run([]string{"--version"}, opts)
	if err == nil || !strings.Contains(err.Error(), "--version") {
		t.Fatalf("expected --version error, got: %v", err)
	}

	err = Run([]string{"-v"}, opts)
	if err == nil || !strings.Contains(err.Error(), "-v") {
		t.Fatalf("expected -v error, got: %v", err)
	}
}
