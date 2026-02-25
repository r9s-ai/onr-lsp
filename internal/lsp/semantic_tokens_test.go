package lsp

import "testing"

func TestSemanticTokensFull_EncodesData(t *testing.T) {
	text := "provider \"openai\" {\n  defaults {\n    request {\n      req_map openai_chat_to_openai_responses;\n    }\n  }\n}\n"
	res := semanticTokensFull(text)
	if len(res.Data) == 0 {
		t.Fatalf("expected semantic token data")
	}
	if len(res.Data)%5 != 0 {
		t.Fatalf("semantic token data should be groups of 5, got len=%d", len(res.Data))
	}
}

func TestClassifyIdentifierType(t *testing.T) {
	if got := classifyIdentifierType("provider", "top", false); got != semanticTypeKeyword {
		t.Fatalf("provider should be keyword, got %d", got)
	}
	if got := classifyIdentifierType("req_map", "request", false); got != semanticTypeProperty {
		t.Fatalf("req_map in request should be property, got %d", got)
	}
	if got := classifyIdentifierType("openai", "request", true); got != semanticTypeEnumMember {
		t.Fatalf("mode token should be enumMember, got %d", got)
	}
	if got := classifyIdentifierType("unknown_token", "request", false); got != -1 {
		t.Fatalf("unknown token should be untyped, got %d", got)
	}
}

func TestLexSemantic_CommentsAndNumbers(t *testing.T) {
	toks := lexSemantic("# c1\n// c2\noauth_timeout_ms 1500;\n")
	if len(toks) == 0 {
		t.Fatalf("expected tokens")
	}
	foundComment := false
	foundNumber := false
	for _, tok := range toks {
		if tok.kind == semanticLexComment {
			foundComment = true
		}
		if tok.kind == semanticLexNumber && tok.text == "1500" {
			foundNumber = true
		}
	}
	if !foundComment {
		t.Fatalf("expected comment token")
	}
	if !foundNumber {
		t.Fatalf("expected number token")
	}
}

func TestEncodeSemanticSpans_Delta(t *testing.T) {
	spans := []semanticSpan{
		{line: 1, start: 2, length: 3, typ: semanticTypeKeyword},
		{line: 1, start: 8, length: 4, typ: semanticTypeProperty},
		{line: 2, start: 1, length: 2, typ: semanticTypeOperator},
	}
	data := encodeSemanticSpans(spans)
	if len(data) != 15 {
		t.Fatalf("expected 15 numbers, got %d", len(data))
	}
	if data[0] != 1 || data[1] != 2 {
		t.Fatalf("first token delta mismatch: %v", data[:5])
	}
	if data[5] != 0 || data[6] != 6 {
		t.Fatalf("second token same-line delta mismatch: %v", data[5:10])
	}
	if data[10] != 1 || data[11] != 1 {
		t.Fatalf("third token next-line delta mismatch: %v", data[10:15])
	}
}
