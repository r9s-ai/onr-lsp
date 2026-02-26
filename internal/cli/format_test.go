package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatStdinToStdout(t *testing.T) {
	t.Parallel()

	in := strings.NewReader("provider \"x\" {\ndefaults {\nrequest {\nreq_map openai_chat_to_openai_responses;\n}\n}\n}\n")
	var out bytes.Buffer
	err := Run([]string{"format"}, Options{
		Stdin:       in,
		Stdout:      &out,
		Stderr:      &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	if err != nil {
		t.Fatalf("run format command: %v", err)
	}

	want := "provider \"x\" {\n  defaults {\n    request {\n      req_map openai_chat_to_openai_responses;\n    }\n  }\n}\n"
	if out.String() != want {
		t.Fatalf("unexpected format output\n--- got ---\n%s\n--- want ---\n%s", out.String(), want)
	}
}

func TestFormatWriteFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "openai.conf")
	in := "provider \"x\" {\ndefaults {\nrequest {\nreq_map openai_chat_to_openai_responses;\n}\n}\n}\n"
	if err := os.WriteFile(path, []byte(in), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	err := Run([]string{"format", "--write", path}, Options{
		Stdin:       strings.NewReader(""),
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	if err != nil {
		t.Fatalf("run format --write command: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formatted file: %v", err)
	}
	want := "provider \"x\" {\n  defaults {\n    request {\n      req_map openai_chat_to_openai_responses;\n    }\n  }\n}\n"
	if string(got) != want {
		t.Fatalf("unexpected file content\n--- got ---\n%s\n--- want ---\n%s", string(got), want)
	}
}

func TestFormatWriteRequiresFilePath(t *testing.T) {
	t.Parallel()

	err := Run([]string{"format", "--write"}, Options{
		Stdin:       strings.NewReader("provider \"x\" {}"),
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "--write requires a file path") {
		t.Fatalf("expected write path error, got: %v", err)
	}
}

func TestFormatRejectsExtraArgs(t *testing.T) {
	t.Parallel()

	err := Run([]string{"format", "a.conf", "b.conf"}, Options{
		Stdin:       strings.NewReader(""),
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		ServeRunner: func(opts ServeRuntimeOptions) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "format accepts at most one file path") {
		t.Fatalf("expected format args error, got: %v", err)
	}
}
