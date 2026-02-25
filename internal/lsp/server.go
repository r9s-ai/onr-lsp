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

	dslconfig "github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
)

var ServerVersion = "dev"

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
	TextDocumentSync       int                    `json:"textDocumentSync"`
	CompletionProvider     *completionProvider    `json:"completionProvider,omitempty"`
	HoverProvider          bool                   `json:"hoverProvider"`
	SemanticTokensProvider *semanticTokensOptions `json:"semanticTokensProvider,omitempty"`
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
	case "textDocument/semanticTokens/full":
		return s.handleSemanticTokensFull(msg.ID, msg.Params)
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
			SemanticTokensProvider: &semanticTokensOptions{
				Legend: semanticTokensLegend{
					TokenTypes:     semanticTokenLegendTypes,
					TokenModifiers: []string{},
				},
				Full: true,
			},
		},
		ServerInfo: serverInfo{
			Name:    "onr-lsp",
			Version: ServerVersion,
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
	block := currentCompletionBlock(text, p.Position)
	doc, ok := dslconfig.DirectiveHoverInBlock(word, block)
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

func (s *Server) handleSemanticTokensFull(id *json.RawMessage, params json.RawMessage) error {
	var p semanticTokensParams
	if err := json.Unmarshal(params, &p); err != nil {
		return s.replyError(id, -32602, "invalid params for semantic tokens")
	}
	text := s.docs[p.TextDocument.URI]
	return s.reply(id, semanticTokensFull(text))
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
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      idVal,
		"result":  result,
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
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      idVal,
		"error": &respError{
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

func complete(text string, pos Position) []CompletionItem {
	line := lineAt(text, pos.Line)
	prefix := line
	if pos.Character >= 0 && pos.Character <= len(line) {
		prefix = line[:pos.Character]
	}
	block := currentCompletionBlock(text, pos)

	if dir, argPrefix, ok := enumArgCompletionPrefix(prefix, block); ok {
		values := enumValuesByDirectiveInBlock(dir, block)
		if len(values) > 0 {
			return completionItemsFromValues(values, argPrefix, dir+" value", "Built-in ONR directive value.", 12)
		}
	}

	dir, dirPrefix, ok := modeCompletionPrefix(prefix)
	if ok && directiveAllowedInPhase(dir, block) {
		return completionItemsFromValues(modeListByDirective(dir), dirPrefix, dir+" mode", "Built-in ONR mapping mode.", 3)
	}

	wordPrefix := currentWordPrefix(prefix)
	dirs := directiveListByBlock(block)
	return completionItemsFromValues(dirs, wordPrefix, "directive", "ONR DSL directive.", 14)
}

func modeCompletionPrefix(linePrefix string) (directive string, prefix string, ok bool) {
	for _, dir := range dslconfig.ModeDirectiveNames() {
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
	if idx > 0 && isWordChar(linePrefix[idx-1]) {
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
	return dslconfig.ModesByDirective(directive)
}

func enumValuesByDirectiveInBlock(directive, block string) []string {
	return dslconfig.DirectiveArgEnumValuesInBlock(directive, block, 0)
}

func directiveAllowedInPhase(directive, phase string) bool {
	allowed := dslconfig.DirectiveAllowedBlocks(directive)
	if len(allowed) == 0 {
		return true
	}
	for _, block := range allowed {
		if block == phase {
			return true
		}
	}
	return false
}

func enumArgCompletionPrefix(linePrefix, block string) (directive string, prefix string, ok bool) {
	for _, dir := range directiveListByBlock(block) {
		if len(enumValuesByDirectiveInBlock(dir, block)) == 0 {
			continue
		}
		if pfx, ok := directiveCompletionPrefix(linePrefix, dir); ok {
			return dir, pfx, true
		}
	}
	return "", "", false
}

func currentCompletionBlock(text string, pos Position) string {
	stack := currentBlockStack(text, pos)
	if len(stack) == 0 {
		return "_top"
	}
	return stack[len(stack)-1]
}

func currentBlockStack(text string, pos Position) []string {
	toks := lex(text)
	stack := make([]string, 0, 8)
	pending := ""
	lockedPending := false

	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tokenAfterPosition(tok, pos) {
			break
		}
		switch tok.kind {
		case tokIdent:
			if isBlockKeyword(tok.text) {
				if tok.text == "match" {
					pending = tok.text
					lockedPending = true
					continue
				}
				if !lockedPending {
					pending = tok.text
				}
			}
		case tokLBrace:
			name := pending
			if name == "" {
				name = "unknown"
			}
			stack = append(stack, name)
			pending = ""
			lockedPending = false
		case tokRBrace:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			pending = ""
			lockedPending = false
		case tokSemicolon:
			if !lockedPending {
				pending = ""
			}
		}
	}
	return stack
}

func isBlockKeyword(s string) bool {
	return dslconfig.IsBlockDirective(s)
}

func tokenAfterPosition(tok token, pos Position) bool {
	if tok.line > pos.Line {
		return true
	}
	if tok.line < pos.Line {
		return false
	}
	return tok.col > pos.Character
}

func currentWordPrefix(linePrefix string) string {
	if linePrefix == "" {
		return ""
	}
	i := len(linePrefix) - 1
	for i >= 0 && isWordChar(linePrefix[i]) {
		i--
	}
	return linePrefix[i+1:]
}

func completionItemsFromValues(values []string, prefix, detail, docs string, kind int) []CompletionItem {
	if len(values) == 0 {
		return nil
	}
	items := make([]CompletionItem, 0, len(values))
	for _, v := range values {
		if prefix != "" && !strings.HasPrefix(v, prefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:         v,
			Kind:          kind,
			Detail:        detail,
			Documentation: docs,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

func directiveListByBlock(block string) []string {
	return dslconfig.DirectivesByBlock(block)
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
