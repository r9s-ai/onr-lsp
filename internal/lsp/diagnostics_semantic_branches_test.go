package lsp

import (
	"errors"
	"strings"
	"testing"

	dslconfig "github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
)

func TestDiagnostics_SyntaxDirectiveErrors(t *testing.T) {
	text := "syntax next-router/0.1\nprovider \"x\" {}\n"
	diags := analyze(text)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for invalid syntax directive")
	}
	msg := diags[0].Message
	if !strings.Contains(msg, "expected syntax version string literal") {
		t.Fatalf("unexpected diagnostic: %v", diags)
	}
}

func TestDiagnostics_SkipStatementLBrace(t *testing.T) {
	text := "provider \"x\" { defaults { request { req_map { bad; } } } }"
	diags := analyze(text)
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "req_map does not use '{ ... }'; expected ';'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected skipStatement lbrace diagnostic, got: %+v", diags)
	}
}

func TestDiagnostics_SkipBalancedBlockMissingRBrace(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    match api = \"chat.completions\" {\n      upstream {\n        set_path \"/v1\";\n"
	diags := analyze(text)
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "missing closing '}'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing closing brace diagnostic, got: %+v", diags)
	}
}

func TestSemanticModeHelpers(t *testing.T) {
	toks := []token{
		{kind: tokIdent, text: "req_map", line: 0, col: 0},
		{kind: tokOther, text: "=", line: 0, col: 7},
		{kind: tokString, text: "\"openai\"", line: 0, col: 9},
		{kind: tokSemicolon, text: ";", line: 0, col: 17},
	}
	modeTok, ok := nextModeToken(toks, 1)
	if !ok {
		t.Fatalf("expected to find next mode token")
	}
	if got := normalizeModeToken(modeTok); got != "openai" {
		t.Fatalf("expected trimmed quoted mode, got %q", got)
	}

	if _, ok := nextModeToken([]token{{kind: tokSemicolon}}, 0); ok {
		t.Fatalf("expected false when next token is semicolon")
	}
	if got := normalizeModeToken(token{kind: tokIdent, text: "  anthropic  "}); got != "anthropic" {
		t.Fatalf("expected trimmed ident mode, got %q", got)
	}
}

func TestAnalyzeSemanticModes_IgnoreNonStatementStartAndMissingMode(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    request {\n      # req_map here in comment should be ignored\n      req_map ;\n      json_set \"$.x\" \"y\"; req_map openai_chat_to_openai_responses;\n    }\n  }\n}\n"
	diags := analyzeSemanticModes(text)
	for _, d := range diags {
		if strings.Contains(d.Message, "unsupported req_map mode") {
			t.Fatalf("did not expect unsupported mode diagnostic here, got: %+v", diags)
		}
	}
}

func TestSemanticDirectivePositionFromMustBeMessage(t *testing.T) {
	text := "syntax \"next-router/0.1\";\nprovider \"x\" {\n  defaults {\n    balance {\n      balance_mode openai;\n      balance_unit EUR;\n    }\n  }\n}\n"
	msg := `provider "x" in "/tmp/x.conf": defaults.balance balance_unit must be USD or CNY`
	line, col, ok := semanticDirectivePosition(text, msg)
	if !ok {
		t.Fatalf("expected semantic directive position")
	}
	if line != 5 {
		t.Fatalf("expected line 5 for balance_unit, got %d", line)
	}
	if col <= 0 {
		t.Fatalf("expected positive column for balance_unit, got %d", col)
	}
}

func TestSemanticDirectivePositionFromUnsupportedModeMessage(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    models {\n      models_mode abc;\n    }\n  }\n}\n"
	msg := `provider "x" in "/tmp/x.conf": defaults.models unsupported models_mode "abc"`
	line, col, ok := semanticDirectivePosition(text, msg)
	if !ok {
		t.Fatalf("expected semantic directive position")
	}
	if line != 3 {
		t.Fatalf("expected line 3 for models_mode, got %d", line)
	}
	if col <= 0 {
		t.Fatalf("expected positive column for models_mode, got %d", col)
	}
}

func TestSemanticDirectivePositionWithScopePrefersBlock(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    balance {\n      method GET;\n    }\n    models {\n      method BAD;\n    }\n  }\n}\n"
	line, col, ok := semanticDirectivePositionWithScope(text, "method", "defaults.models")
	if !ok {
		t.Fatalf("expected semantic directive position with scope")
	}
	if line != 6 {
		t.Fatalf("expected models method at line 6, got %d", line)
	}
	if col <= 0 {
		t.Fatalf("expected positive column, got %d", col)
	}
}

func TestDiagnosticFromStructuredValidationIssue(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    balance {\n      method GET;\n    }\n    models {\n      method BAD;\n    }\n  }\n}\n"
	err := &dslconfig.ValidationIssue{
		Directive: "method",
		Scope:     "defaults.models",
		Err:       errors.New(`provider "x" in "/tmp/x.conf": defaults.models method must be GET or POST`),
	}
	diag := diagnosticFromSemanticMessage(text, err.Error())
	// Fallback path only uses message and may point to first 'method'. ensure structured helper is better:
	line, _, ok := semanticDirectivePositionWithScope(text, err.Directive, err.Scope)
	if !ok {
		t.Fatalf("expected structured position from scope")
	}
	if line != 6 {
		t.Fatalf("expected line 6 for models.method, got %d", line)
	}
	_ = diag
}
