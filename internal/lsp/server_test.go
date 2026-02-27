package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/r9s-ai/open-next-router/onr-core/pkg/dslspec"
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

func TestCompleteReqMapNotInResponsePhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { response { req_map openai_ } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { response { req_map openai_")})
	if len(items) != 0 {
		t.Fatalf("expected no completion items for req_map in response phase, got: %+v", items)
	}
}

func TestCompleteErrorMapModes(t *testing.T) {
	text := "provider \"x\" {\n  defaults { error { error_map o } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { error { error_map o")})
	if len(items) == 0 {
		t.Fatalf("expected completion items, got none")
	}
	found := false
	for _, it := range items {
		if it.Label == "openai" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error_map mode openai")
	}
}

func TestCompleteOAuthModeOnlyInAuthPhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { request { oauth_mode o } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { request { oauth_mode o")})
	if len(items) != 0 {
		t.Fatalf("expected no oauth_mode completion outside auth phase")
	}

	text = "provider \"x\" {\n  defaults { auth { oauth_mode o } }\n}\n"
	items = complete(text, Position{Line: 1, Character: len("  defaults { auth { oauth_mode o")})
	if len(items) == 0 {
		t.Fatalf("expected oauth_mode completion in auth phase")
	}
}

func TestCompleteBalanceModeInBalancePhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { balance { balance_mode o } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { balance { balance_mode o")})
	if len(items) == 0 {
		t.Fatalf("expected balance_mode completion in balance phase")
	}
	found := false
	for _, it := range items {
		if it.Label == "openai" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected openai mode in balance_mode completion, got: %+v", items)
	}
}

func TestCompleteModelsModeInModelsPhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { models { models_mode g } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { models { models_mode g")})
	if len(items) == 0 {
		t.Fatalf("expected models_mode completion in models phase")
	}
	found := false
	for _, it := range items {
		if it.Label == "gemini" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected gemini mode in models_mode completion, got: %+v", items)
	}
}

func TestCompleteBalanceUnitEnumValues(t *testing.T) {
	text := "provider \"x\" {\n  defaults { balance { balance_unit U } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { balance { balance_unit U")})
	if len(items) == 0 {
		t.Fatalf("expected balance_unit enum completion in balance phase")
	}
	found := false
	for _, it := range items {
		if it.Label == "USD" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected USD in balance_unit completion, got: %+v", items)
	}
}

func TestCompleteMethodEnumValuesInModelsPhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { models { method P } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { models { method P")})
	if len(items) == 0 {
		t.Fatalf("expected method enum completion in models phase")
	}
	found := false
	for _, it := range items {
		if it.Label == "POST" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected POST in method completion, got: %+v", items)
	}
}

func TestCompleteOAuthContentTypeEnumValuesInAuthPhase(t *testing.T) {
	text := "provider \"x\" {\n  defaults { auth { oauth_content_type j } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { auth { oauth_content_type j")})
	if len(items) == 0 {
		t.Fatalf("expected oauth_content_type enum completion in auth phase")
	}
	found := false
	for _, it := range items {
		if it.Label == "json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected json in oauth_content_type completion, got: %+v", items)
	}
}

func TestCompleteDirectiveInAuthBlock(t *testing.T) {
	text := "provider \"x\" {\n  defaults { auth { a } }\n}\n"
	items := complete(text, Position{Line: 1, Character: len("  defaults { auth { a")})
	if len(items) == 0 {
		t.Fatalf("expected directive completion items, got none")
	}
	found := false
	for _, it := range items {
		if it.Label == "auth_bearer" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected auth_bearer in auth block completion, got: %+v", items)
	}
}

func TestCompleteDirectiveTopLevel(t *testing.T) {
	text := "s"
	items := complete(text, Position{Line: 0, Character: 1})
	if len(items) == 0 {
		t.Fatalf("expected top-level completion items, got none")
	}
	found := false
	for _, it := range items {
		if it.Label == "syntax" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected syntax in top-level completion, got: %+v", items)
	}
}

func TestHoverDocs(t *testing.T) {
	text := "provider \"x\" { response { sse_parse anthropic_to_openai_chunks; } }"
	word, _ := wordAt(text, Position{Line: 0, Character: 28})
	if word != "sse_parse" {
		t.Fatalf("expected word sse_parse, got %q", word)
	}
	if _, ok := dslspec.DirectiveHover(word); !ok {
		t.Fatalf("expected hover docs for %q", word)
	}
}

func TestHoverDocsForUsageExtract(t *testing.T) {
	text := "provider \"x\" { metrics { usage_extract openai; } }"
	word, _ := wordAt(text, Position{Line: 0, Character: 31})
	if word != "usage_extract" {
		t.Fatalf("expected word usage_extract, got %q", word)
	}
	if _, ok := dslspec.DirectiveHover(word); !ok {
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

func TestDiagnosticsWrongPhaseDirective(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    response {\n      req_map openai_chat_to_anthropic_messages;\n    }\n  }\n}\n"
	diags := analyze(text)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "not allowed in response block") && strings.Contains(d.Message, "quick fix: move it into request") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected wrong-phase directive diagnostic, got: %+v", diags)
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

func TestDiagnosticsTopLevelSyntaxDirective(t *testing.T) {
	text := "syntax \"next-router/0.1\";\nprovider \"x\" {\n  defaults {\n    upstream_config {\n      base_url = \"https://example.com\";\n    }\n  }\n}\n"
	diags := analyze(text)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown top-level directive: syntax") {
			t.Fatalf("syntax directive should be accepted, got diagnostics: %+v", diags)
		}
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

func TestSemanticDiagnosticsMultipleModeErrors(t *testing.T) {
	text := "provider \"x\" {\n  defaults {\n    request { req_map bad_req_mode; }\n    response { resp_map bad_resp_mode; }\n    upstream_config { base_url = \"https://example.com\"; }\n  }\n}\n"
	diags := collectDiagnostics("file:///tmp/x.conf", text)
	if len(diags) < 2 {
		t.Fatalf("expected at least 2 diagnostics, got: %+v", diags)
	}
	reqErr := false
	respErr := false
	for _, d := range diags {
		if strings.Contains(d.Message, "unsupported req_map mode") {
			reqErr = true
		}
		if strings.Contains(d.Message, "unsupported resp_map mode") {
			respErr = true
		}
	}
	if !reqErr || !respErr {
		t.Fatalf("expected both req_map and resp_map semantic diagnostics, got: %+v", diags)
	}
}

func TestReplyIncludesNullResult(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, nil)
	rawID := json.RawMessage("1")
	if err := s.reply(&rawID, nil); err != nil {
		t.Fatalf("reply failed: %v", err)
	}
	payload := out.String()
	if !strings.Contains(payload, "\"result\":null") {
		t.Fatalf("expected result:null in payload, got: %s", payload)
	}
}

func TestHoverRequestWithNoDocReturnsNullResult(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, nil)
	uri := "file:///tmp/a.conf"
	s.docs[uri] = "provider \"x\" {\n  defaults {}\n}\n"

	// choose character on whitespace to force empty/unknown hover.
	params, err := json.Marshal(hoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: 1, Character: 0},
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	rawID := json.RawMessage("2")
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  "textDocument/hover",
		Params:  params,
	}); err != nil {
		t.Fatalf("handle hover: %v", err)
	}

	msg, err := readMessage(bufio.NewReader(bytes.NewReader(out.Bytes())))
	if err != nil {
		t.Fatalf("read LSP response: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["result"]; !ok {
		t.Fatalf("expected result field in response, got: %v", resp)
	}
	if resp["result"] != nil {
		t.Fatalf("expected null result for missing hover docs, got: %#v", resp["result"])
	}
}

func TestHoverUsesBlockSpecificDoc(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, nil)
	uri := "file:///tmp/b.conf"
	s.docs[uri] = "provider \"x\" {\n  defaults {\n    balance {\n      set_header \"Authorization\" \"Bearer x\";\n    }\n  }\n}\n"

	params, err := json.Marshal(hoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: 3, Character: 8}, // set_header
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	rawID := json.RawMessage("3")
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  "textDocument/hover",
		Params:  params,
	}); err != nil {
		t.Fatalf("handle hover: %v", err)
	}

	msg, err := readMessage(bufio.NewReader(bytes.NewReader(out.Bytes())))
	if err != nil {
		t.Fatalf("read LSP response: %v", err)
	}
	var resp struct {
		Result *struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Result == nil {
		t.Fatalf("expected non-nil hover result")
	}
	if !strings.Contains(resp.Result.Contents.Value, "balance query request") {
		t.Fatalf("expected balance-specific hover doc, got: %q", resp.Result.Contents.Value)
	}
}
