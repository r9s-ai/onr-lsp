package lsp

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	dslconfig "github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
	"github.com/r9s-ai/open-next-router/onr-core/pkg/dslspec"
)

var scannerErrRe = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.+)$`)
var providerNameRe = regexp.MustCompile(`(?m)\bprovider\s+"([^"]+)"`)
var semanticDirectiveMustBeRe = regexp.MustCompile(`:\s*[^\s]+\s+([a-z_][a-z0-9_]*)\s+must\s+be\b`)
var semanticUnsupportedModeRe = regexp.MustCompile(`unsupported\s+([a-z_][a-z0-9_]*)\s+mode\b`)
var semanticUnsupportedDirectiveRe = regexp.MustCompile(`unsupported\s+([a-z_][a-z0-9_]*)\b`)

func collectDiagnostics(uri, text string) []Diagnostic {
	out := make([]Diagnostic, 0, 8)
	out = append(out, analyze(text)...)
	out = append(out, analyzeSemanticModes(text)...)
	out = append(out, analyzeSemantic(uri, text)...)
	return dedupeDiagnostics(out)
}

func analyzeSemantic(uri, text string) []Diagnostic {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	p, err := writeTempProviderFile(uri, text)
	if err != nil {
		return []Diagnostic{newDiagnostic(0, 0, "semantic validation setup failed: "+err.Error())}
	}
	defer func() { _ = os.Remove(p) }()

	_, err = dslconfig.ValidateProviderFile(p)
	if err == nil {
		return nil
	}

	msg := err.Error()
	if strings.Contains(msg, "declares provider") && strings.Contains(msg, "expected") {
		// Usually a transient edit-state mismatch, not a semantic DSL problem.
		return nil
	}
	var issue *dslconfig.ValidationIssue
	if errors.As(err, &issue) {
		if line, col, ok := semanticDirectivePositionWithScope(text, issue.Directive, issue.Scope); ok {
			return []Diagnostic{newDiagnostic(line, col, msg)}
		}
	}

	if m := scannerErrRe.FindStringSubmatch(msg); len(m) == 5 {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		return []Diagnostic{newDiagnostic(max(line-1, 0), max(col-1, 0), strings.TrimSpace(m[4]))}
	}
	return []Diagnostic{diagnosticFromSemanticMessage(text, msg)}
}

func analyzeSemanticModes(text string) []Diagnostic {
	toks := lex(text)
	if len(toks) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, 8)
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.kind != tokIdent {
			continue
		}
		if !isStatementStart(toks, i) {
			continue
		}
		allowed := allowedModesForDirective(tok.text)
		if len(allowed) == 0 {
			continue
		}
		modeTok, ok := nextModeToken(toks, i+1)
		if !ok {
			continue
		}
		mode := normalizeModeToken(modeTok)
		if mode == "" {
			continue
		}
		if _, ok := allowed[strings.ToLower(mode)]; ok {
			continue
		}
		out = append(out, newDiagnostic(modeTok.line, modeTok.col, fmt.Sprintf("unsupported %s mode %q", tok.text, mode)))
	}
	return out
}

func isStatementStart(toks []token, idx int) bool {
	if idx == 0 {
		return true
	}
	prev := toks[idx-1]
	switch prev.kind {
	case tokLBrace, tokSemicolon, tokRBrace:
		return true
	default:
		return false
	}
}

func nextModeToken(toks []token, idx int) (token, bool) {
	for i := idx; i < len(toks); i++ {
		switch toks[i].kind {
		case tokIdent, tokString:
			return toks[i], true
		case tokSemicolon, tokLBrace, tokRBrace, tokEOF:
			return token{}, false
		}
	}
	return token{}, false
}

func normalizeModeToken(tok token) string {
	if tok.kind == tokString {
		return strings.Trim(strings.TrimSpace(tok.text), "\"")
	}
	return strings.TrimSpace(tok.text)
}

func allowedModesForDirective(d string) map[string]struct{} {
	return setFromSlice(dslspec.ModesByDirective(d))
}

func setFromSlice(v []string) map[string]struct{} {
	out := make(map[string]struct{}, len(v))
	for _, s := range v {
		out[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
	}
	return out
}

func writeTempProviderFile(uri, content string) (string, error) {
	providerName := extractProviderName(content)
	if providerName == "" {
		providerName = providerNameFromURI(uri)
	}
	if providerName == "" {
		providerName = "untitled"
	}
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	providerName = strings.ReplaceAll(providerName, " ", "-")

	filename := providerName + ".conf"
	p := filepath.Join(os.TempDir(), filename)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write temp provider file: %w", err)
	}
	return p, nil
}

func extractProviderName(content string) string {
	m := providerNameRe.FindStringSubmatch(content)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func providerNameFromURI(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err == nil && u.Scheme == "file" {
		base := path.Base(u.Path)
		return strings.TrimSuffix(base, path.Ext(base))
	}
	base := filepath.Base(uri)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func dedupeDiagnostics(in []Diagnostic) []Diagnostic {
	if len(in) <= 1 {
		return in
	}
	seen := map[string]struct{}{}
	out := make([]Diagnostic, 0, len(in))
	for _, d := range in {
		key := fmt.Sprintf("%d:%d:%s", d.Range.Start.Line, d.Range.Start.Character, d.Message)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, d)
	}
	return out
}

func newDiagnostic(line, col int, msg string) Diagnostic {
	return Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col + 1},
		},
		Severity: 1,
		Source:   "onr-lsp",
		Message:  msg,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func diagnosticFromSemanticMessage(text, msg string) Diagnostic {
	if line, col, ok := semanticDirectivePosition(text, msg); ok {
		return newDiagnostic(line, col, msg)
	}
	return newDiagnostic(0, 0, msg)
}

func semanticDirectivePosition(text, msg string) (int, int, bool) {
	directive := semanticDirectiveFromMessage(msg)
	if directive == "" {
		return 0, 0, false
	}
	return semanticDirectivePositionWithScope(text, directive, "")
}

func semanticDirectivePositionWithScope(text, directive, scope string) (int, int, bool) {
	blockHint := blockHintFromScope(scope)
	toks := lex(text)
	stack := make([]string, 0, 8)
	pending := ""
	lockedPending := false
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		switch tok.kind {
		case tokIdent:
			if isBlockKeyword(tok.text) {
				if tok.text == "match" {
					pending = tok.text
					lockedPending = true
				} else if !lockedPending {
					pending = tok.text
				}
			}
			if tok.text == directive && isStatementStart(toks, i) {
				block := "_top"
				if len(stack) > 0 {
					block = stack[len(stack)-1]
				}
				if blockHint == "" || block == blockHint {
					return tok.line, tok.col, true
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
	return 0, 0, false
}

func blockHintFromScope(scope string) string {
	s := strings.TrimSpace(scope)
	if s == "" {
		return ""
	}
	if strings.Contains(s, ".auth.oauth") {
		return "auth"
	}
	segments := strings.Split(s, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		seg := scopeSegmentBase(segments[i])
		if dslspec.IsBlockDirective(seg) {
			return seg
		}
	}
	return ""
}

func scopeSegmentBase(seg string) string {
	s := strings.TrimSpace(seg)
	if i := strings.IndexByte(s, '['); i >= 0 {
		return s[:i]
	}
	return s
}

func semanticDirectiveFromMessage(msg string) string {
	if m := semanticDirectiveMustBeRe.FindStringSubmatch(strings.ToLower(msg)); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	if m := semanticUnsupportedModeRe.FindStringSubmatch(strings.ToLower(msg)); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	if m := semanticUnsupportedDirectiveRe.FindStringSubmatch(strings.ToLower(msg)); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
