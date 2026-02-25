package lsp

import (
	"strings"
	"testing"
)

func TestDiagnostics_MatchMissingLBrace(t *testing.T) {
	text := "provider \"x\" {\n  match api = \"chat.completions\"\n"
	diags := analyze(text)
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "expected '{' for match block") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing match block brace diagnostic, got: %+v", diags)
	}
}

func TestDiagnostics_UnknownDirectiveWithBlock(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    weird {\n      a;\n    }\n  }\n}\n"
	diags := analyze(text)
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown directive in defaults block: weird") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unknown directive diagnostic, got: %+v", diags)
	}
}
