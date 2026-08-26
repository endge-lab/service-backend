package domainversion

import (
	"encoding/json"
	"strings"

	configurationdomain "github.com/endge-lab/service-backend/internal/domain/configuration"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const defaultActionSource = "defineAction({\n  contract: {\n    input: field('Object'),\n    output: field('Object'),\n  },\n\n  steps: {\n    result: input(),\n  },\n\n  output: output('result'),\n})\n"

// CanonicalizationReport describes compatibility transformations applied while
// producing the current portable domain representation.
type CanonicalizationReport struct {
	IgnoredLegacyFolders       int
	NormalizedFolderReferences int
	MigratedLegacyActions      int
	MigratedLegacyVocabs       int
	SFCEditingDefaultsAdded    bool
}

// Canonicalize returns a detached canonical bundle without mutating the caller's value.
func Canonicalize(bundle entities.PortableBundle) (entities.PortableBundle, CanonicalizationReport, error) {
	raw, err := json.Marshal(bundle)
	if err != nil {
		return entities.PortableBundle{}, CanonicalizationReport{}, err
	}
	var result entities.PortableBundle
	if err = json.Unmarshal(raw, &result); err != nil {
		return entities.PortableBundle{}, CanonicalizationReport{}, err
	}
	report := CanonicalizeInPlace(&result)
	return result, report, nil
}

// CanonicalizeInPlace applies the same deterministic, idempotent representation
// used by export, import, commit, release, backup and status domain versions.
func CanonicalizeInPlace(bundle *entities.PortableBundle) CanonicalizationReport {
	report := CanonicalizationReport{}
	if bundle == nil {
		return report
	}
	if bundle.Documents == nil {
		bundle.Documents = map[string][]map[string]any{}
	}
	if _, exists := bundle.Documents["configurations"]; !exists {
		bundle.Documents["configurations"] = []map[string]any{}
	}
	if bundle.Kind == "" {
		bundle.Kind = "workspace-snapshot"
	}
	if bundle.Workspace == nil {
		bundle.Workspace = map[string]any{}
	}
	delete(bundle.Workspace, "state")
	configurationdomain.RemoveLegacySSE(bundle.Workspace["configuration"])
	if configuration, exists := bundle.Workspace["configuration"]; exists {
		before := canonicalText(configuration)
		bundle.Workspace["configuration"] = configurationdomain.EnsureSFCEditingDefaults(configuration)
		report.SFCEditingDefaultsAdded = before != canonicalText(bundle.Workspace["configuration"])
	}

	if legacy, exists := bundle.Documents["componentSFCs"]; exists {
		bundle.Documents["components"] = append(bundle.Documents["components"], legacy...)
		delete(bundle.Documents, "componentSFCs")
	}
	for _, action := range bundle.Documents["actions"] {
		if strings.TrimSpace(stringValue(action, "source")) != "" {
			continue
		}
		action["source"] = defaultActionSource
		action["sourceVersion"] = float64(1)
		delete(action, "definition")
		delete(action, "input")
		delete(action, "output")
		report.MigratedLegacyActions++
	}
	for _, vocab := range bundle.Documents["vocabs"] {
		if normalizeLegacyVocabSource(vocab) {
			report.MigratedLegacyVocabs++
		}
	}
	for _, folder := range bundle.Documents["folders"] {
		if entityType := stringValue(folder, "entityType"); entityType != "" {
			folder["entityType"] = entities.FolderEntityType(entityType)
		}
		if stringValue(folder, "parentIdentity") == "root-streams" {
			folder["parentIdentity"] = entities.RootFolderIdentity("streams")
		}
	}

	folderTypes := map[string]string{}
	for _, folder := range bundle.Documents["folders"] {
		folderTypes[stringValue(folder, "identity")] = stringValue(folder, "entityType")
	}
	for kind, items := range bundle.Documents {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			delete(item, "state")
			configurationdomain.RemoveLegacySSEFromDocument(kind, item)
			if kind == "queries" && integerValue(item, "sourceVersion") == 1 {
				item["sourceVersion"] = float64(2)
			}
			if kind == "folders" {
				switch stringValue(item, "identity") {
				case "soft-deleted", "no-folder", "root-bindings", "root-streams":
					report.IgnoredLegacyFolders++
					continue
				}
			} else {
				normalizeFolderReference(kind, item, folderTypes, &report)
			}
			filtered = append(filtered, item)
		}
		bundle.Documents[kind] = filtered
	}
	return report
}

func normalizeFolderReference(kind string, item map[string]any, folderTypes map[string]string, report *CanonicalizationReport) {
	folderIdentity := stringValue(item, "folderIdentity")
	if kind == "streams" && folderIdentity == "root-streams" {
		folderIdentity = entities.RootFolderIdentity(kind)
	}
	if folderIdentity != stringValue(item, "folderIdentity") {
		item["folderIdentity"] = folderIdentity
		report.NormalizedFolderReferences++
	}
	switch folderIdentity {
	case "no-folder", "root-bindings":
		delete(item, "folderIdentity")
		report.NormalizedFolderReferences++
	default:
		folderType, hasFolderType := folderTypes[folderIdentity]
		expectedFolderType := entities.FolderEntityType(kind)
		if folderIdentity != "" && ((strings.HasPrefix(folderIdentity, "root-") && folderIdentity != entities.RootFolderIdentity(kind)) || (hasFolderType && folderType != expectedFolderType)) {
			item["folderIdentity"] = entities.RootFolderIdentity(kind)
			report.NormalizedFolderReferences++
		}
	}
}

func normalizeLegacyVocabSource(vocab map[string]any) bool {
	if strings.TrimSpace(stringValue(vocab, "source")) != "" || stringValue(vocab, "mode") != "external_payload" {
		return false
	}
	collection := stringValue(vocab, "collectionSlug")
	if collection == "" {
		collection = stringValue(vocab, "identity")
	}
	authMode := stringValue(vocab, "authMode")
	if authMode != "profile" && authMode != "none" {
		authMode = "inherit"
	}
	auth := `{ mode: ` + sourceString(authMode)
	if authMode == "profile" {
		auth += `, profile: ` + sourceString(stringValue(vocab, "authProfileIdentity"))
	}
	auth += ` }`
	vocab["source"] = "defineVocab({\n  provider: payload({\n    baseUrl: " + legacyVocabBaseURLSource(stringValue(vocab, "baseApiUrl")) + ",\n    collection: " + sourceString(collection) + ",\n    auth: " + auth + ",\n  }),\n\n  outputs: {\n    items: output()\n      .from(response()),\n  },\n})\n"
	vocab["sourceVersion"] = float64(1)
	return true
}

func legacyVocabBaseURLSource(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2 && value[0] == '{' && value[len(value)-1] == '}' {
		name := value[1 : len(value)-1]
		if isSourceEnvironmentName(name) {
			return "env(" + sourceString(name) + ")"
		}
	}
	return sourceString(value)
}

func isSourceEnvironmentName(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || index > 0 && char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func sourceString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func stringValue(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func integerValue(item map[string]any, key string) int {
	switch value := item[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		result, _ := value.Int64()
		return int(result)
	default:
		return 0
	}
}
