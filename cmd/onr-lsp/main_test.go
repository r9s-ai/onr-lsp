package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaybePrintVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	if !maybePrintVersion([]string{"--version"}, buf) {
		t.Fatalf("expected version flag to be handled")
	}
	out := buf.String()
	if !strings.Contains(out, "onr-lsp version=") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestMaybePrintVersion_NoFlag(t *testing.T) {
	buf := &bytes.Buffer{}
	if maybePrintVersion([]string{}, buf) {
		t.Fatalf("expected empty args to not be handled")
	}
	if maybePrintVersion([]string{"serve"}, buf) {
		t.Fatalf("expected non-version arg to not be handled")
	}
}

func TestMaybeRunFormat_NoSubcommand(t *testing.T) {
	handled, err := maybeRunFormat([]string{"serve"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if handled {
		t.Fatalf("expected non-format args to not be handled")
	}
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestMaybeRunFormat_StdinToStdout(t *testing.T) {
	in := strings.NewReader("provider \"x\" {\ndefaults {\nrequest {\nreq_map openai_chat_to_openai_responses;\n}\n}\n}\n")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	handled, err := maybeRunFormat([]string{"format"}, in, out, errOut)
	if !handled {
		t.Fatalf("expected format subcommand to be handled")
	}
	if err != nil {
		t.Fatalf("format should succeed, err=%v stderr=%q", err, errOut.String())
	}
	want := "provider \"x\" {\n  defaults {\n    request {\n      req_map openai_chat_to_openai_responses;\n    }\n  }\n}\n"
	if out.String() != want {
		t.Fatalf("unexpected format output\n--- got ---\n%s\n--- want ---\n%s", out.String(), want)
	}
}

func TestMaybeRunFormat_WriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openai.conf")
	in := "provider \"x\" {\ndefaults {\nrequest {\nreq_map openai_chat_to_openai_responses;\n}\n}\n}\n"
	if err := os.WriteFile(path, []byte(in), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	handled, err := maybeRunFormat([]string{"format", "--write", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !handled {
		t.Fatalf("expected format subcommand to be handled")
	}
	if err != nil {
		t.Fatalf("format --write should succeed, err=%v", err)
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

func TestMaybeRunFormat_WriteRequiresFilePath(t *testing.T) {
	handled, err := maybeRunFormat([]string{"format", "--write"}, strings.NewReader("provider \"x\" {}"), &bytes.Buffer{}, &bytes.Buffer{})
	if !handled {
		t.Fatalf("expected format subcommand to be handled")
	}
	if err == nil || !strings.Contains(err.Error(), "--write requires a file path") {
		t.Fatalf("expected write path error, got: %v", err)
	}
}
