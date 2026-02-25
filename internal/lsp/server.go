package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
)

type Server struct {
	in     *bufio.Reader
	out    io.Writer
	logger *log.Logger

	docs         map[string]string
	shuttingDown bool
}

func NewServer(in io.Reader, out io.Writer, logger *log.Logger) *Server {
	return &Server{
		in:     bufio.NewReader(in),
		out:    out,
		logger: logger,
		docs:   map[string]string{},
	}
}

type inboundMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type responseMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *respError  `json:"error,omitempty"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   serverInfo         `json:"serverInfo"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type serverCapabilities struct {
	TextDocumentSync   int                 `json:"textDocumentSync"`
	CompletionProvider *completionProvider `json:"completionProvider,omitempty"`
	HoverProvider      bool                `json:"hoverProvider"`
}

type completionProvider struct {
	ResolveProvider   bool     `json:"resolveProvider"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type textDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type versionedTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type textDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type didChangeParams struct {
	TextDocument   versionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []textDocumentContentChangeEvent `json:"contentChanges"`
}

type completionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type hoverParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type CompletionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
}

func (s *Server) Run() error {
	for {
		raw, err := readMessage(s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var msg inboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			s.logger.Printf("invalid JSON-RPC payload: %v", err)
			continue
		}

		if msg.Method == "" {
			continue
		}
		if err := s.handle(msg); err != nil {
			s.logger.Printf("handle method=%s error: %v", msg.Method, err)
		}
	}
}

func (s *Server) handle(msg inboundMessage) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg.ID)
	case "initialized":
		return nil
	case "shutdown":
		s.shuttingDown = true
		return s.reply(msg.ID, map[string]any{})
	case "exit":
		if s.shuttingDown {
			return io.EOF
		}
		return io.EOF
	case "textDocument/didOpen":
		var p didOpenParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return err
		}
		s.docs[p.TextDocument.URI] = p.TextDocument.Text
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/didChange":
		var p didChangeParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return err
		}
		if len(p.ContentChanges) == 0 {
			return nil
		}
		s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/completion":
		return s.handleCompletion(msg.ID, msg.Params)
	case "textDocument/hover":
		return s.handleHover(msg.ID, msg.Params)
	default:
		if msg.ID != nil {
			return s.reply(msg.ID, nil)
		}
		return nil
	}
}

func (s *Server) handleInitialize(id *json.RawMessage) error {
	res := initializeResult{
		Capabilities: serverCapabilities{
			TextDocumentSync: 1,
			CompletionProvider: &completionProvider{
				ResolveProvider:   false,
				TriggerCharacters: []string{" ", "_"},
			},
			HoverProvider: true,
		},
		ServerInfo: serverInfo{
			Name:    "onr-lsp",
			Version: "0.1.0",
		},
	}
	return s.reply(id, res)
}

func (s *Server) handleCompletion(id *json.RawMessage, params json.RawMessage) error {
	var p completionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return s.replyError(id, -32602, "invalid params for completion")
	}
	text := s.docs[p.TextDocument.URI]
	items := complete(text, p.Position)
	return s.reply(id, items)
}

func (s *Server) handleHover(id *json.RawMessage, params json.RawMessage) error {
	var p hoverParams
	if err := json.Unmarshal(params, &p); err != nil {
		return s.replyError(id, -32602, "invalid params for hover")
	}
	text := s.docs[p.TextDocument.URI]
	word, rng := wordAt(text, p.Position)
	if word == "" {
		return s.reply(id, nil)
	}

	doc, ok := hoverDocs[word]
	if !ok {
		return s.reply(id, nil)
	}
	h := Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: doc,
		},
		Range: &rng,
	}
	return s.reply(id, h)
}

func (s *Server) publishDiagnostics(uri string) error {
	text, ok := s.docs[uri]
	if !ok {
		return nil
	}
	diags := collectDiagnostics(uri, text)
	params := publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	}
	return s.notify("textDocument/publishDiagnostics", params)
}

func (s *Server) reply(id *json.RawMessage, result interface{}) error {
	if id == nil {
		return nil
	}
	var idVal interface{}
	if err := json.Unmarshal(*id, &idVal); err != nil {
		idVal = string(*id)
	}
	resp := responseMessage{
		JSONRPC: "2.0",
		ID:      idVal,
		Result:  result,
	}
	return writeMessage(s.out, resp)
}

func (s *Server) replyError(id *json.RawMessage, code int, msg string) error {
	if id == nil {
		return nil
	}
	var idVal interface{}
	if err := json.Unmarshal(*id, &idVal); err != nil {
		idVal = string(*id)
	}
	resp := responseMessage{
		JSONRPC: "2.0",
		ID:      idVal,
		Error: &respError{
			Code:    code,
			Message: msg,
		},
	}
	return writeMessage(s.out, resp)
}

func (s *Server) notify(method string, params interface{}) error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return writeMessage(s.out, payload)
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length %q: %w", v, err)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeMessage(w io.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(body))
	return err
}

var reqMapModes = []string{
	"openai_chat_to_openai_responses",
	"openai_chat_to_anthropic_messages",
	"openai_chat_to_gemini_generate_content",
	"anthropic_to_openai_chat",
	"gemini_to_openai_chat",
}

var respMapModes = []string{
	"openai_responses_to_openai_chat",
	"anthropic_to_openai_chat",
	"gemini_to_openai_chat",
	"openai_to_anthropic_messages",
	"openai_to_gemini_chat",
	"openai_to_gemini_generate_content",
}

var sseParseModes = []string{
	"openai_responses_to_openai_chat_chunks",
	"anthropic_to_openai_chunks",
	"openai_to_anthropic_chunks",
	"openai_to_gemini_chunks",
	"gemini_to_openai_chat_chunks",
}

func complete(text string, pos Position) []CompletionItem {
	line := lineAt(text, pos.Line)
	prefix := line
	if pos.Character >= 0 && pos.Character <= len(line) {
		prefix = line[:pos.Character]
	}
	dir, dirPrefix, ok := modeCompletionPrefix(prefix)
	if !ok {
		return nil
	}

	modes := modeListByDirective(dir)
	items := make([]CompletionItem, 0, len(modes))
	for _, mode := range modes {
		if dirPrefix != "" && !strings.HasPrefix(mode, dirPrefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:         mode,
			Kind:          3,
			Detail:        dir + " mode",
			Documentation: "Built-in ONR mapping mode.",
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

func modeCompletionPrefix(linePrefix string) (directive string, prefix string, ok bool) {
	for _, dir := range []string{"req_map", "resp_map", "sse_parse"} {
		if pfx, ok := directiveCompletionPrefix(linePrefix, dir); ok {
			return dir, pfx, true
		}
	}
	return "", "", false
}

func directiveCompletionPrefix(linePrefix, directive string) (string, bool) {
	idx := strings.LastIndex(linePrefix, directive)
	if idx < 0 {
		return "", false
	}
	after := linePrefix[idx+len(directive):]
	if after == "" {
		return "", true
	}
	if !strings.HasPrefix(after, " ") && !strings.HasPrefix(after, "\t") {
		return "", false
	}
	return strings.TrimSpace(after), true
}

func modeListByDirective(directive string) []string {
	switch directive {
	case "req_map":
		return reqMapModes
	case "resp_map":
		return respMapModes
	case "sse_parse":
		return sseParseModes
	default:
		return nil
	}
}

var hoverDocs = map[string]string{
	"provider":        "`provider \"name\" { ... }`\n\nDefines one provider DSL block. File name should match provider name.",
	"defaults":        "`defaults { ... }`\n\nDefault phases shared by all `match` rules unless overridden.",
	"match":           "`match api = \"...\" [stream = true|false] { ... }`\n\nRoute rule. First match wins.",
	"upstream_config": "`upstream_config { base_url = \"...\"; }`\n\nProvider-level upstream base URL config.",
	"auth":            "`auth { ... }`\n\nAuthentication directives for upstream requests.",
	"request":         "`request { ... }`\n\nRequest rewrite/transform directives.",
	"upstream":        "`upstream { ... }`\n\nUpstream path/query routing directives.",
	"response":        "`response { ... }`\n\nDownstream response mapping/transformation directives.",
	"error":           "`error { error_map <mode>; }`\n\nNormalize upstream error payloads.",
	"metrics":         "`metrics { ... }`\n\nToken usage and finish reason extraction rules.",
	"req_map":         "`req_map <mode>;`\n\nMap request JSON between API schemas.",
	"resp_map":        "`resp_map <mode>;`\n\nMap non-stream response JSON.",
	"sse_parse":       "`sse_parse <mode>;`\n\nMap streaming SSE events/chunks.",
}

func wordAt(text string, pos Position) (string, Range) {
	line := lineAt(text, pos.Line)
	if line == "" {
		return "", Range{}
	}
	ch := pos.Character
	if ch < 0 {
		ch = 0
	}
	if ch > len(line) {
		ch = len(line)
	}
	left := ch
	for left > 0 && isWordChar(line[left-1]) {
		left--
	}
	right := ch
	for right < len(line) && isWordChar(line[right]) {
		right++
	}
	if left == right {
		return "", Range{}
	}
	return line[left:right], Range{
		Start: Position{Line: pos.Line, Character: left},
		End:   Position{Line: pos.Line, Character: right},
	}
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '.'
}

func lineAt(text string, line int) string {
	if line < 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return lines[line]
}
