package lsp

import "strings"

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokLBrace
	tokRBrace
	tokSemicolon
	tokOther
)

type token struct {
	kind tokenKind
	text string
	line int
	col  int
}

type parser struct {
	tokens []token
	i      int
	diags  []Diagnostic
}

func analyze(text string) []Diagnostic {
	p := &parser{tokens: lex(text)}
	p.parseFile()
	return p.diags
}

func (p *parser) parseFile() {
	for {
		tok := p.peek()
		if tok.kind == tokEOF {
			return
		}
		if tok.kind != tokIdent {
			p.next()
			continue
		}
		if tok.text == "syntax" {
			p.next() // syntax
			v := p.next()
			if v.kind != tokString {
				p.add(v, "expected syntax version string literal")
			}
			semi := p.next()
			if semi.kind != tokSemicolon {
				p.add(semi, "expected ';' after syntax directive")
			}
			continue
		}
		if tok.text != "provider" {
			p.add(tok, "unknown top-level directive: "+tok.text)
			p.skipStmtOrBlock()
			continue
		}
		p.next() // provider
		name := p.next()
		if name.kind != tokString {
			p.add(name, "expected provider name string literal")
		}
		lb := p.next()
		if lb.kind != tokLBrace {
			p.add(lb, "expected '{' after provider name")
			continue
		}
		p.parseBlock("provider", providerDirectives)
	}
}

func (p *parser) parseBlock(name string, directives map[string]directiveSpec) {
	for {
		tok := p.peek()
		switch tok.kind {
		case tokEOF:
			p.addPos(tok.line, tok.col, "missing closing '}' for "+name+" block")
			return
		case tokRBrace:
			p.next()
			return
		case tokIdent:
			spec, ok := directives[tok.text]
			if !ok {
				if allowed := allowedBlocksForDirective(tok.text); len(allowed) > 0 {
					p.add(tok, "directive "+tok.text+" is not allowed in "+name+" block; allowed in: "+strings.Join(allowed, ", ")+"; quick fix: move it into "+allowed[0]+" { ... }")
				} else {
					p.add(tok, "unknown directive in "+name+" block: "+tok.text)
				}
				p.skipStmtOrBlock()
				continue
			}
			p.next()
			if spec.block {
				if spec.parseHeaderUntilBrace {
					if !p.skipUntilLBrace(spec.name) {
						return
					}
				} else {
					lb := p.next()
					if lb.kind != tokLBrace {
						p.add(lb, "expected '{' after "+spec.name)
						continue
					}
				}
				if spec.sub != nil {
					p.parseBlock(spec.name, spec.sub)
				} else {
					p.skipBalancedBlock(spec.name)
				}
				continue
			}
			p.skipStatement(spec.name)
		default:
			p.next()
		}
	}
}

func (p *parser) skipUntilLBrace(name string) bool {
	for {
		tok := p.next()
		switch tok.kind {
		case tokLBrace:
			return true
		case tokEOF:
			p.add(tok, "expected '{' for "+name+" block")
			return false
		}
	}
}

func (p *parser) skipBalancedBlock(name string) {
	depth := 1
	for depth > 0 {
		tok := p.next()
		if tok.kind == tokEOF {
			p.add(tok, "missing closing '}' for "+name+" block")
			return
		}
		if tok.kind == tokLBrace {
			depth++
		}
		if tok.kind == tokRBrace {
			depth--
		}
	}
}

func (p *parser) skipStatement(name string) {
	for {
		tok := p.next()
		switch tok.kind {
		case tokSemicolon:
			return
		case tokLBrace:
			p.add(tok, name+" does not use '{ ... }'; expected ';'")
			p.skipBalancedBlock(name)
			return
		case tokRBrace, tokEOF:
			return
		}
	}
}

func (p *parser) skipStmtOrBlock() {
	tok := p.peek()
	if tok.kind == tokLBrace {
		p.next()
		p.skipBalancedBlock("unknown")
		return
	}
	for {
		tok := p.next()
		switch tok.kind {
		case tokSemicolon:
			return
		case tokLBrace:
			p.skipBalancedBlock("unknown")
			return
		case tokRBrace, tokEOF:
			return
		}
	}
}

func (p *parser) next() token {
	if p.i >= len(p.tokens) {
		return token{kind: tokEOF}
	}
	t := p.tokens[p.i]
	p.i++
	return t
}

func (p *parser) peek() token {
	if p.i >= len(p.tokens) {
		return token{kind: tokEOF}
	}
	return p.tokens[p.i]
}

func (p *parser) add(tok token, msg string) {
	p.addPos(tok.line, tok.col, msg)
}

func (p *parser) addPos(line, col int, msg string) {
	p.diags = append(p.diags, Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col + 1},
		},
		Severity: 1,
		Source:   "onr-lsp",
		Message:  msg,
	})
}

type directiveSpec struct {
	name                  string
	block                 bool
	parseHeaderUntilBrace bool
	sub                   map[string]directiveSpec
}

var providerDirectives = map[string]directiveSpec{
	"defaults": {name: "defaults", block: true, sub: phaseDirectivesDefaults},
	"match":    {name: "match", block: true, parseHeaderUntilBrace: true, sub: phaseDirectivesMatch},
}

var phaseDirectivesDefaults = map[string]directiveSpec{
	"upstream_config": {name: "upstream_config", block: true, sub: upstreamConfigDirectives},
	"auth":            {name: "auth", block: true, sub: authDirectives},
	"request":         {name: "request", block: true, sub: requestDirectives},
	"response":        {name: "response", block: true, sub: responseDirectives},
	"error":           {name: "error", block: true, sub: errorDirectives},
	"metrics":         {name: "metrics", block: true, sub: metricsDirectives},
	"balance":         {name: "balance", block: true},
	"models":          {name: "models", block: true},
}

var phaseDirectivesMatch = map[string]directiveSpec{
	"upstream": {name: "upstream", block: true, sub: upstreamDirectives},
	"auth":     {name: "auth", block: true, sub: authDirectives},
	"request":  {name: "request", block: true, sub: requestDirectives},
	"response": {name: "response", block: true, sub: responseDirectives},
	"error":    {name: "error", block: true, sub: errorDirectives},
	"metrics":  {name: "metrics", block: true, sub: metricsDirectives},
}

var upstreamConfigDirectives = map[string]directiveSpec{
	"base_url": {name: "base_url"},
}

var upstreamDirectives = map[string]directiveSpec{
	"set_path":  {name: "set_path"},
	"set_query": {name: "set_query"},
	"del_query": {name: "del_query"},
}

var authDirectives = map[string]directiveSpec{
	"auth_bearer":            {name: "auth_bearer"},
	"auth_header_key":        {name: "auth_header_key"},
	"auth_oauth_bearer":      {name: "auth_oauth_bearer"},
	"oauth_mode":             {name: "oauth_mode"},
	"oauth_token_url":        {name: "oauth_token_url"},
	"oauth_client_id":        {name: "oauth_client_id"},
	"oauth_client_secret":    {name: "oauth_client_secret"},
	"oauth_refresh_token":    {name: "oauth_refresh_token"},
	"oauth_scope":            {name: "oauth_scope"},
	"oauth_audience":         {name: "oauth_audience"},
	"oauth_method":           {name: "oauth_method"},
	"oauth_content_type":     {name: "oauth_content_type"},
	"oauth_token_path":       {name: "oauth_token_path"},
	"oauth_expires_in_path":  {name: "oauth_expires_in_path"},
	"oauth_token_type_path":  {name: "oauth_token_type_path"},
	"oauth_timeout_ms":       {name: "oauth_timeout_ms"},
	"oauth_refresh_skew_sec": {name: "oauth_refresh_skew_sec"},
	"oauth_fallback_ttl_sec": {name: "oauth_fallback_ttl_sec"},
	"oauth_form":             {name: "oauth_form"},
}

var requestDirectives = map[string]directiveSpec{
	"set_header":         {name: "set_header"},
	"del_header":         {name: "del_header"},
	"model_map":          {name: "model_map"},
	"model_map_default":  {name: "model_map_default"},
	"json_set":           {name: "json_set"},
	"json_set_if_absent": {name: "json_set_if_absent"},
	"json_del":           {name: "json_del"},
	"json_rename":        {name: "json_rename"},
	"req_map":            {name: "req_map"},
}

var responseDirectives = map[string]directiveSpec{
	"resp_passthrough":   {name: "resp_passthrough"},
	"resp_map":           {name: "resp_map"},
	"sse_parse":          {name: "sse_parse"},
	"json_set":           {name: "json_set"},
	"json_set_if_absent": {name: "json_set_if_absent"},
	"json_del":           {name: "json_del"},
	"json_rename":        {name: "json_rename"},
	"sse_json_del_if":    {name: "sse_json_del_if"},
}

var errorDirectives = map[string]directiveSpec{
	"error_map": {name: "error_map"},
}

var metricsDirectives = map[string]directiveSpec{
	"usage_extract":           {name: "usage_extract"},
	"input_tokens":            {name: "input_tokens"},
	"output_tokens":           {name: "output_tokens"},
	"cache_read_tokens":       {name: "cache_read_tokens"},
	"cache_write_tokens":      {name: "cache_write_tokens"},
	"total_tokens":            {name: "total_tokens"},
	"input_tokens_path":       {name: "input_tokens_path"},
	"output_tokens_path":      {name: "output_tokens_path"},
	"cache_read_tokens_path":  {name: "cache_read_tokens_path"},
	"cache_write_tokens_path": {name: "cache_write_tokens_path"},
	"finish_reason_extract":   {name: "finish_reason_extract"},
	"finish_reason_path":      {name: "finish_reason_path"},
}

func allowedBlocksForDirective(d string) []string {
	switch d {
	case "upstream_config":
		return []string{"defaults"}
	case "upstream":
		return []string{"match"}
	case "auth_bearer", "auth_header_key", "auth_oauth_bearer", "oauth_mode", "oauth_token_url", "oauth_client_id", "oauth_client_secret", "oauth_refresh_token", "oauth_scope", "oauth_audience", "oauth_method", "oauth_content_type", "oauth_token_path", "oauth_expires_in_path", "oauth_token_type_path", "oauth_timeout_ms", "oauth_refresh_skew_sec", "oauth_fallback_ttl_sec", "oauth_form":
		return []string{"auth"}
	case "set_header", "del_header", "model_map", "model_map_default", "req_map":
		return []string{"request"}
	case "resp_passthrough", "resp_map", "sse_parse", "sse_json_del_if":
		return []string{"response"}
	case "error_map":
		return []string{"error"}
	case "usage_extract", "input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens", "total_tokens", "input_tokens_path", "output_tokens_path", "cache_read_tokens_path", "cache_write_tokens_path", "finish_reason_extract", "finish_reason_path":
		return []string{"metrics"}
	case "set_path", "set_query", "del_query":
		return []string{"upstream"}
	case "base_url":
		return []string{"upstream_config"}
	case "provider", "defaults", "match", "auth", "request", "response", "error", "metrics", "balance", "models":
		// block keywords are handled by parser; keep unknown behavior where syntax doesn't match.
		return nil
	default:
		return nil
	}
}

func lex(input string) []token {
	var out []token
	line, col := 0, 0

	emit := func(kind tokenKind, text string, l, c int) {
		out = append(out, token{kind: kind, text: text, line: l, col: c})
	}

	for i := 0; i < len(input); {
		ch := input[i]
		if ch == '\n' {
			line++
			col = 0
			i++
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\r' {
			col++
			i++
			continue
		}
		if ch == '#' {
			for i < len(input) && input[i] != '\n' {
				i++
				col++
			}
			continue
		}
		if ch == '/' && i+1 < len(input) && input[i+1] == '/' {
			for i < len(input) && input[i] != '\n' {
				i++
				col++
			}
			continue
		}

		startLine, startCol := line, col
		switch ch {
		case '{':
			emit(tokLBrace, "{", startLine, startCol)
			i++
			col++
			continue
		case '}':
			emit(tokRBrace, "}", startLine, startCol)
			i++
			col++
			continue
		case ';':
			emit(tokSemicolon, ";", startLine, startCol)
			i++
			col++
			continue
		case '"':
			j := i + 1
			c2 := col + 1
			for j < len(input) {
				if input[j] == '\\' && j+1 < len(input) {
					j += 2
					c2 += 2
					continue
				}
				if input[j] == '"' {
					j++
					c2++
					break
				}
				if input[j] == '\n' {
					break
				}
				j++
				c2++
			}
			emit(tokString, input[i:j], startLine, startCol)
			col = c2
			i = j
			continue
		default:
			if isIdentStart(ch) {
				j := i + 1
				for j < len(input) && isIdentPart(input[j]) {
					j++
				}
				txt := input[i:j]
				emit(tokIdent, txt, startLine, startCol)
				col += j - i
				i = j
				continue
			}
			emit(tokOther, string(ch), startLine, startCol)
			i++
			col++
		}
	}
	out = append(out, token{kind: tokEOF, line: line, col: col})
	return out
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentPart(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9') || strings.ContainsRune(".-", rune(b))
}
