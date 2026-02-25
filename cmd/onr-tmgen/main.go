package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dslconfig "github.com/r9s-ai/open-next-router/onr-core/pkg/dslconfig"
)

type tmLanguage struct {
	Schema    string                 `json:"$schema"`
	Name      string                 `json:"name"`
	ScopeName string                 `json:"scopeName"`
	Patterns  []map[string]string    `json:"patterns"`
	Repo      map[string]interface{} `json:"repository"`
}

type tmPattern struct {
	Name  string `json:"name,omitempty"`
	Match string `json:"match,omitempty"`
	Begin string `json:"begin,omitempty"`
	End   string `json:"end,omitempty"`

	Patterns []tmPattern `json:"patterns,omitempty"`
}

type tmRepositoryEntry struct {
	Patterns []tmPattern `json:"patterns"`
}

var output = flag.String("output", "vscode/syntaxes/onr.tmLanguage.json", "output grammar file path")

func main() {
	flag.Parse()

	meta := dslconfig.DirectiveMetadataList()
	if len(meta) == 0 {
		fatalf("directive metadata is empty")
	}

	directiveSet := map[string]struct{}{}
	modeDirectiveSet := map[string]struct{}{}
	for _, m := range meta {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		if isKeyword(name) {
			continue
		}
		directiveSet[name] = struct{}{}
		if len(m.Modes) > 0 {
			modeDirectiveSet[name] = struct{}{}
		}
	}

	directives := setToSortedSlice(directiveSet)
	modeDirectives := setToSortedSlice(modeDirectiveSet)
	keywords := []string{
		"syntax", "provider", "defaults", "match", "upstream_config", "upstream",
		"auth", "request", "response", "error", "metrics", "balance", "models",
		"true", "false",
	}

	g := tmLanguage{
		Schema:    "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json",
		Name:      "ONR DSL",
		ScopeName: "source.onr-dsl",
		Patterns: []map[string]string{
			{"include": "#comments"},
			{"include": "#provider-name"},
			{"include": "#mode-value"},
			{"include": "#keywords"},
			{"include": "#directives"},
			{"include": "#numbers"},
			{"include": "#strings"},
			{"include": "#operators"},
		},
		Repo: map[string]interface{}{
			"comments": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "comment.line.number-sign.onr-dsl", Match: "#.*$"},
				{Name: "comment.line.double-slash.onr-dsl", Match: "//.*$"},
			}},
			"provider-name": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "entity.name.namespace.onr-dsl", Match: `(?<=\bprovider\s+)"[^"\n]*"`},
			}},
			"mode-value": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "constant.other.mode.onr-dsl", Match: modeValueMatch(modeDirectives, false)},
				{Name: "constant.other.mode.onr-dsl", Match: modeValueMatch(modeDirectives, true)},
			}},
			"keywords": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "keyword.control.onr-dsl", Match: wordRegex(keywords)},
			}},
			"directives": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "support.type.property-name.onr-dsl", Match: wordRegex(directives)},
			}},
			"numbers": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "constant.numeric.onr-dsl", Match: `\b\d+(?:\.\d+)?\b`},
			}},
			"strings": tmRepositoryEntry{Patterns: []tmPattern{
				{
					Name:  "string.quoted.double.onr-dsl",
					Begin: `"`,
					End:   `"`,
					Patterns: []tmPattern{
						{Name: "constant.character.escape.onr-dsl", Match: `\\.`},
					},
				},
			}},
			"operators": tmRepositoryEntry{Patterns: []tmPattern{
				{Name: "keyword.operator.onr-dsl", Match: `[=;{}]`},
			}},
		},
	}

	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		fatalf("marshal grammar: %v", err)
	}
	b := []byte(sb.String())

	outPath := *output
	if !filepath.IsAbs(outPath) {
		wd, err := os.Getwd()
		if err != nil {
			fatalf("getwd: %v", err)
		}
		outPath = filepath.Join(wd, outPath)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		fatalf("write grammar: %v", err)
	}
}

func modeValueMatch(modeDirectives []string, quoted bool) string {
	base := `(?<=\b(?:` + joinRegexAlternation(modeDirectives) + `)\s+)`
	if quoted {
		return base + `"[^"\n]*"`
	}
	return base + `[A-Za-z_][A-Za-z0-9_\.-]*`
}

func wordRegex(words []string) string {
	if len(words) == 0 {
		return `\b\B`
	}
	return `\b(?:` + joinRegexAlternation(words) + `)\b`
}

func joinRegexAlternation(words []string) string {
	escaped := make([]string, 0, len(words))
	for _, w := range words {
		escaped = append(escaped, regexpQuoteMeta(w))
	}
	return strings.Join(escaped, "|")
}

func setToSortedSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func regexpQuoteMeta(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`.`, `\\.`,
		`+`, `\\+`,
		`*`, `\\*`,
		`?`, `\\?`,
		`(`, `\\(`,
		`)`, `\\)`,
		`[`, `\\[`,
		`]`, `\\]`,
		`{`, `\\{`,
		`}`, `\\}`,
		`^`, `\\^`,
		`$`, `\\$`,
		`|`, `\\|`,
	)
	return replacer.Replace(s)
}

func isKeyword(s string) bool {
	switch s {
	case "syntax", "provider", "defaults", "match", "upstream_config", "upstream", "auth", "request", "response", "error", "metrics", "balance", "models", "true", "false":
		return true
	default:
		return false
	}
}

func fatalf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "onr-tmgen: "+format+"\n", args...)
	os.Exit(1)
}
