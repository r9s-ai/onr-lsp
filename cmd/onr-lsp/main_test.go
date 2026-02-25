package main

import (
	"bytes"
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
