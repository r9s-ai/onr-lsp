package lsp

import (
	"strings"
	"testing"
)

func TestCompleteReqMapModes(t *testing.T) {
	text := "provider \"x\" {\n  defaults { request { req_map op } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { request { req_map op")})
	if len(items) == 0 {
		t.Fatalf("expected completion items, got none")
	}
	found := false
	for _, it := range items {
		if it.Label == "openai_chat_to_openai_responses" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected req_map built-in mode in completion list")
	}
}

func TestCompleteRespMapModes(t *testing.T) {
	text := "provider \"x\" {\n  defaults { response { resp_map openai_ } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { response { resp_map openai_")})
	if len(items) == 0 {
		t.Fatalf("expected completion items, got none")
	}
	for _, it := range items {
		if strings.HasPrefix(it.Label, "openai_") {
			return
		}
	}
	t.Fatalf("expected resp_map mode with openai_ prefix")
}

func TestCompleteSSEParseModes(t *testing.T) {
	text := "provider \"x\" {\n  defaults { response { sse_parse anthropic_ } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { response { sse_parse anthropic_")})
	if len(items) == 0 {
		t.Fatalf("expected completion items, got none")
	}
	found := false
	for _, it := range items {
		if it.Label == "anthropic_to_openai_chunks" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sse_parse mode anthropic_to_openai_chunks")
	}
}

func TestHoverDocs(t *testing.T) {
	text := "provider \"x\" { response { sse_parse anthropic_to_openai_chunks; } }"
	word, _ := wordAt(text, Position{Line: 0, Character: 28})
	if word != "sse_parse" {
		t.Fatalf("expected word sse_parse, got %q", word)
	}
	if _, ok := hoverDocs[word]; !ok {
		t.Fatalf("expected hover docs for %q", word)
	}
}

func TestDiagnosticsUnknownDirective(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    request {\n      req_map openai_chat_to_anthropic_messages;\n      bad_cmd foo;\n    }\n  }\n}\n"
	diags := analyze(text)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	ok := false
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown directive") {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected unknown directive diagnostic, got: %+v", diags)
	}
}

func TestDiagnosticsMissingBrace(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    request {\n      req_map openai_chat_to_anthropic_messages;\n  }\n}\n"
	diags := analyze(text)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	ok := false
	for _, d := range diags {
		if strings.Contains(d.Message, "missing closing '}'") {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected missing brace diagnostic, got: %+v", diags)
	}
}

func TestSemanticDiagnosticsUnsupportedReqMapMode(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    upstream_config {\n      base_url = \"https://example.com\";\n    }\n    request {\n      req_map not_a_real_mapper;\n    }\n  }\n}\n"
	diags := collectDiagnostics("file:///tmp/x.conf", text)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "unsupported req_map mode") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected semantic diagnostic for unsupported req_map mode, got: %+v", diags)
	}
}
