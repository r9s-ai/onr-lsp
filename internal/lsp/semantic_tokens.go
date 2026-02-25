package lsp

import (
	"strings"

	dslconfig "github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
)

const (
	semanticTypeKeyword = iota
	semanticTypeString
	semanticTypeNumber
	semanticTypeComment
	semanticTypeOperator
	semanticTypeNamespace
	semanticTypeProperty
	semanticTypeEnumMember
)

var semanticTokenLegendTypes = []string{
	"keyword",
	"string",
	"number",
	"comment",
	"operator",
	"namespace",
	"property",
	"enumMember",
}

type semanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type semanticTokensOptions struct {
	Legend semanticTokensLegend `json:"legend"`
	Full   bool                 `json:"full"`
}

type semanticTokens struct {
	Data []uint32 `json:"data"`
}

type semanticTokensParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type semanticLexKind int

const (
	semanticLexIdent semanticLexKind = iota
	semanticLexString
	semanticLexNumber
	semanticLexComment
	semanticLexOperator
	semanticLexLBrace
	semanticLexRBrace
	semanticLexSemicolon
	semanticLexOther
)

type semanticLexToken struct {
	kind   semanticLexKind
	text   string
	line   int
	col    int
	length int
}

type semanticSpan struct {
	line   int
	start  int
	length int
	typ    int
}

const (
	prevSigBOF = iota
	prevSigLBrace
	prevSigRBrace
	prevSigSemicolon
	prevSigOther
)

func semanticTokensFull(text string) semanticTokens {
	spans := classifySemanticSpans(text)
	return semanticTokens{Data: encodeSemanticSpans(spans)}
}

func classifySemanticSpans(text string) []semanticSpan {
	toks := lexSemantic(text)
	if len(toks) == 0 {
		return nil
	}

	spans := make([]semanticSpan, 0, len(toks))
	stack := make([]string, 0, 8)
	pending := ""
	lockedPending := false
	currentDirective := ""
	modePending := false
	prevSig := prevSigBOF

	for _, tok := range toks {
		switch tok.kind {
		case semanticLexComment:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeComment})
			continue
		case semanticLexLBrace:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeOperator})
			name := pending
			if name == "" {
				name = "unknown"
			}
			stack = append(stack, name)
			pending = ""
			lockedPending = false
			currentDirective = ""
			modePending = false
			prevSig = prevSigLBrace
			continue
		case semanticLexRBrace:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeOperator})
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			pending = ""
			lockedPending = false
			currentDirective = ""
			modePending = false
			prevSig = prevSigRBrace
			continue
		case semanticLexSemicolon:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeOperator})
			if !lockedPending {
				pending = ""
			}
			currentDirective = ""
			modePending = false
			prevSig = prevSigSemicolon
			continue
		case semanticLexOperator:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeOperator})
			modePending = false
			prevSig = prevSigOther
			continue
		}

		block := "top"
		if len(stack) > 0 {
			block = stack[len(stack)-1]
		}
		statementStart := prevSig == prevSigBOF || prevSig == prevSigLBrace || prevSig == prevSigRBrace || prevSig == prevSigSemicolon

		switch tok.kind {
		case semanticLexString:
			tokenType := semanticTypeString
			if currentDirective == "provider" {
				tokenType = semanticTypeNamespace
			} else if modePending {
				tokenType = semanticTypeEnumMember
				modePending = false
			}
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: tokenType})
			prevSig = prevSigOther
		case semanticLexNumber:
			spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: semanticTypeNumber})
			prevSig = prevSigOther
		case semanticLexIdent:
			tokenType := classifyIdentifierType(tok.text, block, modePending)
			if tokenType >= 0 {
				spans = append(spans, semanticSpan{line: tok.line, start: tok.col, length: tok.length, typ: tokenType})
			}
			if modePending {
				modePending = false
			}
			if isBlockKeyword(tok.text) {
				if tok.text == "match" {
					pending = tok.text
					lockedPending = true
				} else if !lockedPending {
					pending = tok.text
				}
			}
			if statementStart {
				currentDirective = tok.text
				modePending = len(dslconfig.ModesByDirective(tok.text)) > 0
			}
			prevSig = prevSigOther
		default:
			prevSig = prevSigOther
		}
	}

	return spans
}

func classifyIdentifierType(word, block string, modePending bool) int {
	w := strings.TrimSpace(word)
	if w == "" {
		return -1
	}
	if modePending {
		return semanticTypeEnumMember
	}
	if w == "true" || w == "false" || w == "syntax" {
		return semanticTypeKeyword
	}
	if isBlockKeyword(w) {
		return semanticTypeKeyword
	}
	if directiveInBlock(w, block) {
		return semanticTypeProperty
	}
	return -1
}

func directiveInBlock(name, block string) bool {
	for _, d := range dslconfig.DirectivesByBlock(block) {
		if d == name {
			return true
		}
	}
	return false
}

func encodeSemanticSpans(spans []semanticSpan) []uint32 {
	if len(spans) == 0 {
		return nil
	}
	data := make([]uint32, 0, len(spans)*5)
	prevLine := 0
	prevStart := 0
	for i, s := range spans {
		lineDelta := s.line
		startDelta := s.start
		if i > 0 {
			lineDelta = s.line - prevLine
			if lineDelta == 0 {
				startDelta = s.start - prevStart
			}
		}
		data = append(data, uint32(lineDelta), uint32(startDelta), uint32(s.length), uint32(s.typ), 0)
		prevLine = s.line
		prevStart = s.start
	}
	return data
}

func lexSemantic(input string) []semanticLexToken {
	out := make([]semanticLexToken, 0, len(input)/3)
	line, col := 0, 0

	emit := func(kind semanticLexKind, text string, l, c int) {
		out = append(out, semanticLexToken{
			kind:   kind,
			text:   text,
			line:   l,
			col:    c,
			length: len(text),
		})
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

		startLine, startCol := line, col
		if ch == '#' || (ch == '/' && i+1 < len(input) && input[i+1] == '/') {
			j := i
			for j < len(input) && input[j] != '\n' {
				j++
			}
			emit(semanticLexComment, input[i:j], startLine, startCol)
			col += j - i
			i = j
			continue
		}

		switch ch {
		case '{':
			emit(semanticLexLBrace, "{", startLine, startCol)
			i++
			col++
			continue
		case '}':
			emit(semanticLexRBrace, "}", startLine, startCol)
			i++
			col++
			continue
		case ';':
			emit(semanticLexSemicolon, ";", startLine, startCol)
			i++
			col++
			continue
		case '=':
			emit(semanticLexOperator, "=", startLine, startCol)
			i++
			col++
			continue
		case '"':
			j := i + 1
			for j < len(input) {
				if input[j] == '\\' && j+1 < len(input) {
					j += 2
					continue
				}
				if input[j] == '"' {
					j++
					break
				}
				if input[j] == '\n' {
					break
				}
				j++
			}
			emit(semanticLexString, input[i:j], startLine, startCol)
			col += j - i
			i = j
			continue
		}

		if isIdentStart(ch) {
			j := i + 1
			for j < len(input) && isIdentPart(input[j]) {
				j++
			}
			emit(semanticLexIdent, input[i:j], startLine, startCol)
			col += j - i
			i = j
			continue
		}
		if ch >= '0' && ch <= '9' {
			j := i + 1
			for j < len(input) && input[j] >= '0' && input[j] <= '9' {
				j++
			}
			if j < len(input) && input[j] == '.' {
				k := j + 1
				for k < len(input) && input[k] >= '0' && input[k] <= '9' {
					k++
				}
				if k > j+1 {
					j = k
				}
			}
			emit(semanticLexNumber, input[i:j], startLine, startCol)
			col += j - i
			i = j
			continue
		}

		emit(semanticLexOther, string(ch), startLine, startCol)
		i++
		col++
	}
	return out
}
