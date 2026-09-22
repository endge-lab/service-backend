package access

import (
	"encoding/json"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPointerEscapingArraysAndTypedEquals(t *testing.T) {
	path, ok := ParsePointer("/a~1b/~0flag/0")
	if !ok {
		t.Fatal("valid pointer rejected")
	}
	policy := NewPolicy("p", "source", "v", []Rule{{Path: path, Equals: json.Number("9007199254740993"), Grant: entities.MappedAccessGrant{Scope: "platform", Role: "admin"}}})
	claims := map[string]any{"a/b": map[string]any{"~flag": []any{json.Number("9007199254740993")}}}
	if len(policy.Map(claims)) != 1 {
		t.Fatal("escaped array pointer did not match exact integer")
	}
	claims["a/b"] = map[string]any{"~flag": []any{json.Number("9007199254740992")}}
	if len(policy.Map(claims)) != 0 {
		t.Fatal("neighbor integer incorrectly matched")
	}
	claims["a/b"] = map[string]any{"~flag": []any{"9007199254740993"}}
	if len(policy.Map(claims)) != 0 {
		t.Fatal("string matched a numeric condition")
	}
	if !equalNumbers("1e2", "100.0") {
		t.Fatal("equivalent numeric scalars differ")
	}
	for _, path := range []string{"not/a/pointer", "/~", "/~3"} {
		if _, ok := ParsePointer(path); ok {
			t.Fatalf("accepted %s", path)
		}
	}
}

func TestLocalManagementPolicy(t *testing.T) {
	var policy *Policy
	if policy.External() || policy.RequireLocal() != nil || policy.Management().Mode != "local" {
		t.Fatal("nil policy is not local")
	}
	if NewPolicy("p", "s", "v", nil).RequireLocal() == nil {
		t.Fatal("external policy permits manual writes")
	}
}
