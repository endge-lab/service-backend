// Package access owns external access mapping; transport and YAML parsing stay outside it.
package access

import (
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

type Rule struct {
	Path     []string
	Equals   any
	Contains *string
	Grant    entities.MappedAccessGrant
}

// Policy is compiled once at startup. A nil policy means local management.
type Policy struct {
	provider string
	name     string
	version  string
	rules    []Rule
}

func NewPolicy(provider, name, version string, rules []Rule) *Policy {
	return &Policy{provider: provider, name: name, version: version, rules: rules}
}
func (p *Policy) External() bool { return p != nil }
func (p *Policy) Provider() string {
	if p == nil {
		return ""
	}
	return p.provider
}
func (p *Policy) Version() string {
	if p == nil {
		return ""
	}
	return p.version
}
func (p *Policy) Management() entities.AccessManagement {
	if p == nil {
		return entities.AccessManagement{Mode: "local"}
	}
	return entities.AccessManagement{Mode: "external", SourceName: p.name}
}
func (p *Policy) RequireLocal() error {
	if p != nil {
		return domainerrors.Forbidden("access_managed_externally", "Права доступа управляются внешней системой")
	}
	return nil
}

func (p *Policy) Map(claims map[string]any) []entities.MappedAccessGrant {
	result := make([]entities.MappedAccessGrant, 0)
	if p == nil {
		return result
	}
	grants := map[string]entities.MappedAccessGrant{}
	for _, rule := range p.rules {
		value, ok := pointerValue(claims, rule.Path)
		if !ok || !matches(value, rule) {
			continue
		}
		key := rule.Grant.Scope + "/" + rule.Grant.Workspace
		if previous, exists := grants[key]; !exists || rank(rule.Grant.Role) > rank(previous.Role) {
			grants[key] = rule.Grant
		}
	}
	keys := make([]string, 0, len(grants))
	for key := range grants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, grants[key])
	}
	return result
}

func pointerValue(value any, path []string) (any, bool) {
	for _, segment := range path {
		switch current := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = current[segment]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(current) || strconv.Itoa(index) != segment {
				return nil, false
			}
			value = current[index]
		default:
			return nil, false
		}
	}
	return value, value != nil
}

func matches(value any, rule Rule) bool {
	if rule.Contains != nil {
		values, ok := value.([]any)
		if !ok {
			return false
		}
		found := false
		for _, item := range values {
			text, ok := item.(string)
			if !ok {
				return false
			}
			found = found || text == *rule.Contains
		}
		return found
	}
	switch expected := rule.Equals.(type) {
	case bool:
		actual, ok := value.(bool)
		return ok && actual == expected
	case string:
		actual, ok := value.(string)
		return ok && actual == expected
	case json.Number:
		actual, ok := value.(json.Number)
		if !ok {
			return false
		}
		// JSON numbers are compared numerically without losing integer precision.
		return equalNumbers(string(actual), string(expected))
	default:
		return false
	}
}

func rank(role string) int {
	switch role {
	case "admin":
		return 3
	case "editor":
		return 2
	case "viewer":
		return 1
	default:
		return 0
	}
}

// ParsePointer implements the escaping of RFC 6901, without wildcards.
func ParsePointer(path string) ([]string, bool) {
	if !strings.HasPrefix(path, "/") {
		return nil, false
	}
	parts := strings.Split(path[1:], "/")
	for i, part := range parts {
		for j := 0; j < len(part); j++ {
			if part[j] == '~' {
				j++
				if j >= len(part) || (part[j] != '0' && part[j] != '1') {
					return nil, false
				}
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, true
}

func equalNumbers(left, right string) bool {
	a, ok := new(big.Rat).SetString(left)
	if !ok {
		return false
	}
	b, ok := new(big.Rat).SetString(right)
	return ok && a.Cmp(b) == 0
}
