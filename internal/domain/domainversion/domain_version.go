// Package domainversion computes a portable identity for committed domain content.
package domainversion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const prefix = "dv1:sha256:"

type canonicalBundle struct {
	Kind          string                      `json:"kind"`
	SchemaVersion int                         `json:"schemaVersion"`
	Workspace     map[string]any              `json:"workspace"`
	Documents     map[string][]map[string]any `json:"documents"`
}

// Compute returns a stable identity for the part of a bundle that import applies.
// Target-local workspace identity, integrations, credentials and provenance are
// deliberately outside this contract.
func Compute(bundle entities.PortableBundle) (string, error) {
	workspace := portableWorkspace(bundle.Workspace)

	documents := make(map[string][]map[string]any, len(bundle.Documents))
	for kind, values := range bundle.Documents {
		items := make([]map[string]any, 0, len(values))
		for _, value := range values {
			item, cloneErr := cloneMap(value)
			if cloneErr != nil {
				return "", fmt.Errorf("clone %s document for domain version: %w", kind, cloneErr)
			}
			delete(item, "state")
			items = append(items, item)
		}
		sort.SliceStable(items, func(left, right int) bool {
			leftIdentity := text(items[left]["identity"])
			rightIdentity := text(items[right]["identity"])
			if leftIdentity != rightIdentity {
				return leftIdentity < rightIdentity
			}
			return canonicalText(items[left]) < canonicalText(items[right])
		})
		documents[kind] = items
	}

	raw, err := json.Marshal(canonicalBundle{
		Kind:          bundle.Kind,
		SchemaVersion: bundle.SchemaVersion,
		Workspace:     workspace,
		Documents:     documents,
	})
	if err != nil {
		return "", fmt.Errorf("marshal canonical domain: %w", err)
	}
	sum := sha256.Sum256(raw)
	return prefix + hex.EncodeToString(sum[:]), nil
}

// portableWorkspace оставляет только поля Workspace, которые применяет import.
func portableWorkspace(source map[string]any) map[string]any {
	result := map[string]any{}
	for _, key := range []string{"displayName", "description", "dataMode", "configuration", "meta", "active"} {
		if value, exists := source[key]; exists {
			result[key] = value
		}
	}
	return result
}

// ComputeRaw computes a version from a serialized portable bundle.
func ComputeRaw(raw json.RawMessage) (string, error) {
	var bundle entities.PortableBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return "", fmt.Errorf("decode portable bundle for domain version: %w", err)
	}
	return Compute(bundle)
}

// Attach computes and assigns the version without including the field in its own digest.
func Attach(bundle *entities.PortableBundle) error {
	if bundle == nil {
		return fmt.Errorf("portable bundle is required")
	}
	value, err := Compute(*bundle)
	if err != nil {
		return err
	}
	bundle.DomainVersion = value
	return nil
}

func cloneMap(value map[string]any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func canonicalText(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func text(value any) string {
	result, _ := value.(string)
	return result
}
