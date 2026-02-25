package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
)

func TestRun_InitializeShutdownExit(t *testing.T) {
	var in bytes.Buffer
	writeLSPMessage(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	writeLSPMessage(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "shutdown",
		"params":  map[string]any{},
	})
	writeLSPMessage(&in, map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
		"params":  map[string]any{},
	})

	var out bytes.Buffer
	s := NewServer(&in, &out, log.New(io.Discard, "", 0))
	if err := s.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 responses, got %d", len(msgs))
	}
	if msgs[0]["id"] == nil || msgs[0]["result"] == nil {
		t.Fatalf("initialize response missing id/result: %+v", msgs[0])
	}
	if msgs[1]["id"] == nil || msgs[1]["result"] == nil {
		t.Fatalf("shutdown response missing id/result: %+v", msgs[1])
	}
}

func TestHandle_DidOpenPublishesDiagnostics(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(stringsReader(""), &out, log.New(io.Discard, "", 0))

	params, err := json.Marshal(didOpenParams{
		TextDocument: textDocumentItem{
			URI:  "file:///tmp/a.conf",
			Text: "unknown_top foo;",
		},
	})
	if err != nil {
		t.Fatalf("marshal didOpen params: %v", err)
	}
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  params,
	}); err != nil {
		t.Fatalf("handle didOpen: %v", err)
	}

	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) == 0 {
		t.Fatalf("expected diagnostics notification")
	}
	if msgs[0]["method"] != "textDocument/publishDiagnostics" {
		t.Fatalf("expected publishDiagnostics, got: %+v", msgs[0])
	}
}

func TestHandle_InvalidHoverParamsReplyError(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(stringsReader(""), &out, log.New(io.Discard, "", 0))

	rawID := json.RawMessage("7")
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  "textDocument/hover",
		Params:  json.RawMessage(`{"oops":`), // malformed JSON
	}); err != nil {
		t.Fatalf("handle hover should not return error: %v", err)
	}

	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 {
		t.Fatalf("expected one response, got %d", len(msgs))
	}
	if msgs[0]["error"] == nil {
		t.Fatalf("expected error response, got: %+v", msgs[0])
	}
}

func TestHandle_CompletionReturnsItems(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(stringsReader(""), &out, log.New(io.Discard, "", 0))
	uri := "file:///tmp/c.conf"
	s.docs[uri] = "provider \"x\" {\n  defaults { request { req_map o } }\n}\n"

	params, err := json.Marshal(completionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: 1, Character: len("  defaults { request { req_map o")},
	})
	if err != nil {
		t.Fatalf("marshal completion params: %v", err)
	}

	rawID := json.RawMessage("8")
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  "textDocument/completion",
		Params:  params,
	}); err != nil {
		t.Fatalf("handle completion: %v", err)
	}

	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 {
		t.Fatalf("expected one completion response, got %d", len(msgs))
	}
	if msgs[0]["error"] != nil {
		t.Fatalf("expected completion result, got error: %+v", msgs[0]["error"])
	}
	if msgs[0]["result"] == nil {
		t.Fatalf("expected completion result items, got nil")
	}
}

func TestHandle_InvalidCompletionParamsReplyError(t *testing.T) {
	var out bytes.Buffer
	s := NewServer(stringsReader(""), &out, log.New(io.Discard, "", 0))
	rawID := json.RawMessage("9")
	if err := s.handle(inboundMessage{
		JSONRPC: "2.0",
		ID:      &rawID,
		Method:  "textDocument/completion",
		Params:  json.RawMessage(`{"bad":`),
	}); err != nil {
		t.Fatalf("handle completion should not return error: %v", err)
	}

	msgs := readAllLSPMessages(t, out.Bytes())
	if len(msgs) != 1 {
		t.Fatalf("expected one response, got %d", len(msgs))
	}
	if msgs[0]["error"] == nil {
		t.Fatalf("expected error response for invalid completion params")
	}
}

func writeLSPMessage(w *bytes.Buffer, payload any) {
	b, _ := json.Marshal(payload)
	_, _ = w.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(b)))
	_, _ = w.Write(b)
}

func readAllLSPMessages(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(raw))
	out := make([]map[string]any, 0, 4)
	for {
		msg, err := readMessage(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("readMessage: %v", err)
		}
		var obj map[string]any
		if err := json.Unmarshal(msg, &obj); err != nil {
			t.Fatalf("unmarshal LSP message: %v", err)
		}
		out = append(out, obj)
	}
	return out
}

func stringsReader(s string) *bytes.Reader { return bytes.NewReader([]byte(s)) }
