package releases

import (
	"encoding/json"
	"errors"
	"maps"
	"strings"
)

type releaseProgram struct {
	Version               int                        `json:"version"`
	ExportScope           string                     `json:"exportScope"`
	ExecutionGraphVersion int                        `json:"executionGraphVersion"`
	ProgramID             string                     `json:"programId"`
	CompilerVersion       string                     `json:"compilerVersion"`
	Context               json.RawMessage            `json:"context"`
	Artifacts             map[string]json.RawMessage `json:"artifacts"`
	Catalog               json.RawMessage            `json:"catalog"`
}

type releaseDelivery struct {
	Variants []releaseProgram `json:"variants"`
	Modules  json.RawMessage  `json:"modules"`
}

type moduleReference struct {
	Identity string `json:"identity"`
	Version  string `json:"version"`
}

func validateDeliveryPrograms(primary *releaseProgram, delivery *releaseDelivery) ([]map[string]string, error) {
	if delivery.Variants == nil {
		return nil, errors.New("Delivery variants must be an array")
	}
	programs := append([]releaseProgram{*primary}, delivery.Variants...)
	facets := make([]map[string]string, 0, len(programs))
	seen := make(map[string]bool)
	required := make(map[moduleReference]bool)
	workspace := ""
	for _, program := range programs {
		var context struct {
			Workspace     string            `json:"workspace"`
			Facets        map[string]string `json:"facets"`
			Configuration struct {
				DefaultRenderers map[string]moduleReference `json:"defaultRenderers"`
			} `json:"configuration"`
		}
		var catalog struct {
			Workspace struct {
				Identity string `json:"identity"`
				Startup  string `json:"startupCompositionIdentity"`
			} `json:"workspace"`
		}
		if program.Version != 1 || program.ExportScope != "startup-composition" || program.ExecutionGraphVersion != 1 ||
			program.ProgramID == "" || program.CompilerVersion != primary.CompilerVersion || program.Artifacts == nil ||
			json.Unmarshal(program.Context, &context) != nil || context.Workspace == "" || context.Facets == nil ||
			json.Unmarshal(program.Catalog, &catalog) != nil || catalog.Workspace.Identity != context.Workspace || catalog.Workspace.Startup == "" {
			return nil, errors.New("Invalid delivery variant")
		}
		if workspace == "" {
			workspace = context.Workspace
		}
		if context.Workspace != workspace || (len(facets) > 0 && (!maps.Equal(facetKeys(context.Facets), facetKeys(facets[0])) || context.Facets["tenant"] != facets[0]["tenant"])) {
			return nil, errors.New("Delivery variants must belong to the same workspace and tenant")
		}
		key, _ := json.Marshal(context.Facets)
		if seen[string(key)] {
			return nil, errors.New("Duplicate delivery context")
		}
		seen[string(key)] = true
		facets = append(facets, context.Facets)
		for _, reference := range context.Configuration.DefaultRenderers {
			required[reference] = true
		}
		for identity, raw := range program.Artifacts {
			if err := collectDeliveryResourceModules(identity, raw, required); err != nil {
				return nil, err
			}
		}
	}
	if len(delivery.Modules) > 0 {
		var modules []struct {
			Manifest struct {
				moduleReference
				FormatVersion int `json:"formatVersion"`
				HostAPI       int `json:"hostApi"`
			} `json:"manifest"`
			JS string `json:"js"`
		}
		if json.Unmarshal(delivery.Modules, &modules) != nil || modules == nil {
			return nil, errors.New("Invalid embedded modules")
		}
		for _, entry := range modules {
			reference := entry.Manifest.moduleReference
			if !required[reference] || reference.Identity == "" || reference.Version == "" || entry.JS == "" ||
				entry.Manifest.FormatVersion != 1 || entry.Manifest.HostAPI != 1 {
				return nil, errors.New("Invalid, duplicate or unrelated embedded module")
			}
			delete(required, reference)
		}
		if len(required) != 0 {
			return nil, errors.New("Missing embedded modules")
		}
	}
	return facets, nil
}

func facetKeys(facets map[string]string) map[string]bool {
	keys := make(map[string]bool, len(facets))
	for facet := range facets {
		keys[facet] = true
	}
	return keys
}

func collectDeliveryResourceModules(key string, raw json.RawMessage, required map[moduleReference]bool) error {
	if !strings.HasPrefix(key, "composition:") {
		return nil
	}
	var artifact struct {
		Payload struct {
			Resources []struct {
				moduleReference
				Kind string `json:"kind"`
			} `json:"resources"`
		} `json:"payload"`
	}
	if json.Unmarshal(raw, &artifact) != nil || artifact.Payload.Resources == nil {
		return errors.New("Invalid module resources in delivery composition")
	}
	for _, resource := range artifact.Payload.Resources {
		if resource.Kind == "module" {
			required[resource.moduleReference] = true
		}
	}
	return nil
}
