package lsp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
)

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// splitErrWriter succeeds once (header), then fails (body).
type splitErrWriter struct{ calls int }

func (w *splitErrWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 1 {
		return len(p), nil
	}
	return 0, errors.New("body write failed")
}

func TestHandle_InitializedAndExitWithoutShutdown(t *testing.T) {
	s := NewServer(strings.NewReader(""), io.Discard, log.New(io.Discard, "", 0))

	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "initialized"}); err != nil {
		t.Fatalf("initialized should be no-op: %v", err)
	}
	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "exit"}); !errors.Is(err, io.EOF) {
		t.Fatalf("exit should return EOF, got: %v", err)
	}
}

func TestHandle_UnknownMethod(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, log.New(io.Discard, "", 0))

	// With ID -> reply with result:null
	rawID := json.RawMessage("10")
	if err := s.handle(inboundMessage{JSONRPC: "2.0", ID: &rawID, Method: "custom/method"}); err != nil {
		t.Fatalf("handle unknown with id: %v", err)
	}
	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 {
		t.Fatalf("expected one reply for unknown request, got %d", len(msgs))
	}
	if _, ok := msgs[0]["result"]; !ok {
		t.Fatalf("expected result field for unknown request: %+v", msgs[0])
	}

	// Without ID -> notification style, no output.
	out.Reset()
	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "custom/notify"}); err != nil {
		t.Fatalf("handle unknown notify: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output for unknown notification, got %d bytes", out.Len())
	}
}

func TestHandle_DidChangeBranches(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, log.New(io.Discard, "", 0))

	// Invalid params should return error.
	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "textDocument/didChange", Params: json.RawMessage(`{"oops":`)}); err == nil {
		t.Fatalf("expected unmarshal error for malformed didChange params")
	}

	uri := "file:///tmp/change.conf"
	s.docs[uri] = "provider \"x\" {}"
	params, err := json.Marshal(didChangeParams{
		TextDocument: versionedTextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("marshal didChange empty: %v", err)
	}

	// Empty changes -> no diagnostics publish.
	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "textDocument/didChange", Params: params}); err != nil {
		t.Fatalf("didChange with empty changes should be no-op: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output for empty didChange changes, got %d bytes", out.Len())
	}

	params, err = json.Marshal(didChangeParams{
		TextDocument:   versionedTextDocumentIdentifier{URI: uri},
		ContentChanges: []textDocumentContentChangeEvent{{Text: "provider \"x\" { unknown; }"}},
	})
	if err != nil {
		t.Fatalf("marshal didChange: %v", err)
	}
	if err := s.handle(inboundMessage{JSONRPC: "2.0", Method: "textDocument/didChange", Params: params}); err != nil {
		t.Fatalf("didChange should publish diagnostics: %v", err)
	}
	if got := s.docs[uri]; !strings.Contains(got, "unknown") {
		t.Fatalf("expected doc updated from last content change, got: %q", got)
	}
	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 || msgs[0]["method"] != "textDocument/publishDiagnostics" {
		t.Fatalf("expected one diagnostics publish after didChange, got: %+v", msgs)
	}
}

func TestRun_InvalidJSONPayloadContinues(t *testing.T) {
	var in bytes.Buffer
	writeLSPMessage(&in, json.RawMessage(`{`)) // malformed JSON payload
	writeLSPMessage(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      11,
		"method":  "initialize",
		"params":  map[string]any{},
	})

	var out bytes.Buffer
	s := NewServer(&in, &out, log.New(io.Discard, "", 0))
	if err := s.Run(); err != nil {
		t.Fatalf("Run should ignore invalid JSON and continue, got: %v", err)
	}
	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 || msgs[0]["id"] == nil {
		t.Fatalf("expected initialize response after malformed payload, got: %+v", msgs)
	}
}

func TestReplyAndReplyError_NilIDNoOutput(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(strings.NewReader(""), &out, nil)
	if err := s.reply(nil, map[string]any{"ok": true}); err != nil {
		t.Fatalf("reply nil id should no-op: %v", err)
	}
	if err := s.replyError(nil, -32600, "bad request"); err != nil {
		t.Fatalf("replyError nil id should no-op: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output when id is nil, got %d bytes", out.Len())
	}
}

func TestWriteMessage_ErrorPaths(t *testing.T) {
	if err := writeMessage(io.Discard, map[string]any{"bad": func() {}}); err == nil {
		t.Fatalf("expected marshal error")
	}

	if err := writeMessage(errWriter{}, map[string]any{"ok": true}); err == nil {
		t.Fatalf("expected header write error")
	}

	w := &splitErrWriter{}
	if err := writeMessage(w, map[string]any{"ok": true}); err == nil {
		t.Fatalf("expected body write error")
	}
}

func TestCompletionAndWordHelpers_Branches(t *testing.T) {
	if got, ok := directiveCompletionPrefix("xreq_map o", "req_map"); ok || got != "" {
		t.Fatalf("expected false when directive is not word-boundary, got ok=%v got=%q", ok, got)
	}
	if got, ok := directiveCompletionPrefix("req_mapx", "req_map"); ok || got != "" {
		t.Fatalf("expected false when token suffix is not whitespace separated, got ok=%v got=%q", ok, got)
	}
	if got, ok := directiveCompletionPrefix("req_map\top", "req_map"); !ok || got != "op" {
		t.Fatalf("expected tab-separated prefix, got ok=%v got=%q", ok, got)
	}

	if directiveAllowedInPhase("unknown_mode_directive", "request") != true {
		t.Fatalf("unknown directive should default to allowed")
	}
	if got := lineAt("a\nb", -1); got != "" {
		t.Fatalf("lineAt negative should return empty, got %q", got)
	}
	if got := lineAt("a\nb", 9); got != "" {
		t.Fatalf("lineAt out-of-range should return empty, got %q", got)
	}

	if dir, pfx, ok := enumArgCompletionPrefix("balance_unit U", "balance"); !ok || dir != "balance_unit" || pfx != "U" {
		t.Fatalf("expected enum arg completion prefix for balance_unit, got dir=%q pfx=%q ok=%v", dir, pfx, ok)
	}
	if dir, _, ok := enumArgCompletionPrefix("req_map op", "request"); ok || dir != "" {
		t.Fatalf("req_map should be handled by mode completion, not enum args")
	}
}

func TestCurrentBlockStack_Branches(t *testing.T) {
	text := "provider \"x\" {\n  match api = \"chat.completions\" stream = true {\n    response {\n      s\n    }\n  }\n}\n"
	stack := currentBlockStack(text, Position{Line: 3, Character: 6})
	if len(stack) < 3 {
		t.Fatalf("expected nested stack, got: %+v", stack)
	}
	if stack[1] != "match" {
		t.Fatalf("expected second stack element to stay match, got: %+v", stack)
	}

	// LBrace without pending keyword should use "unknown".
	unknown := currentBlockStack("{\n", Position{Line: 0, Character: 1})
	if len(unknown) != 1 || unknown[0] != "unknown" {
		t.Fatalf("expected unknown stack for bare '{', got: %+v", unknown)
	}
}
