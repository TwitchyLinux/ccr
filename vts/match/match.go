// Package match implements filename matching and rewriting.
package match

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/gobwas/glob"
	"go.starlark.net/starlark"
)

// MatchRule contains a single filename match rule and action.
type MatchRule struct {
	P   glob.Glob
	Out OutputMapper
}

// FilenameRules contains a set of filename matching/rewriting rules.
type FilenameRules struct {
	Rules []MatchRule
}

func (m *FilenameRules) RollupHash() []byte {
	h := sha256.New()
	fmt.Fprintln(h, "Output mappings:")
	for _, r := range m.Rules {
		fmt.Fprintf(h, "%v: %v\n", r.P, r.Out)
	}
	return h.Sum(nil)
}

// Match returns the new filename, or the empty string if no match
// is found.
func (m *FilenameRules) Match(artifactPath string) string {
	for _, r := range m.Rules {
		if r.P.Match(artifactPath) {
			return r.Out.Map(artifactPath)
		}
	}
	return ""
}

func BuildFilenameMappers(output *starlark.Dict) (*FilenameRules, error) {
	if output == nil {
		return &FilenameRules{}, nil
	}
	out := &FilenameRules{Rules: make([]MatchRule, output.Len())}

	keys := make([]string, 0, output.Len())
	for i, e := range output.Keys() {
		s, ok := e.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("index %d: key is %T, need string", i, e)
		}
		keys = append(keys, string(s))
	}
	sort.Strings(keys)

	for i, k := range keys {
		v, _, _ := output.Get(starlark.String(k))
		var mapper OutputMapper
		switch m := v.(type) {
		case starlark.String:
			mapper = LiteralOutputMapper(m)
		case OutputMapper:
			mapper = m
		default:
			return nil, fmt.Errorf("key %q: value is %T, need string or mapper", k, v)
		}
		out.Rules[i] = MatchRule{P: glob.MustCompile(strings.TrimPrefix(k, "/")), Out: mapper}
	}
	return out, nil
}
