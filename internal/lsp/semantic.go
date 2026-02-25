package lsp

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
)

var scannerErrRe = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.+)$`)
var providerNameRe = regexp.MustCompile(`(?m)\bprovider\s+"([^"]+)"`)

func collectDiagnostics(uri, text string) []Diagnostic {
	out := make([]Diagnostic, 0, 8)
	out = append(out, analyze(text)...)
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

	if m := scannerErrRe.FindStringSubmatch(msg); len(m) == 5 {
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		return []Diagnostic{newDiagnostic(max(line-1, 0), max(col-1, 0), strings.TrimSpace(m[4]))}
	}
	return []Diagnostic{newDiagnostic(0, 0, msg)}
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
