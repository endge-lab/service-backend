package user_state

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type Scope struct {
	Partition   string                `json:"partition"`
	Environment string                `json:"environment"`
	Facets      map[string]string     `json:"facets"`
	Driver      string                `json:"driver"`
	Source      string                `json:"source"`
	Labels      map[string]FacetLabel `json:"labels"`
}

type FacetLabel struct {
	Facet    string `json:"facet"`
	Document string `json:"document"`
}

type driverConfig struct {
	UserStateDriver string `json:"userStateDriver"`
}

type facetConfig struct {
	Configuration struct {
		Mode  string       `json:"mode"`
		Value driverConfig `json:"value"`
		Patch struct {
			Driver struct {
				Operation string `json:"op"`
				Value     string `json:"value"`
			} `json:"userStateDriver"`
		} `json:"patch"`
	} `json:"configuration"`
}

func (s *UseCase) resolveScope(ctx context.Context, workspace entities.Workspace, selections map[string]string) (entities.UserStateScope, Scope, error) {
	invalid := domainerrors.InvalidInput("user_state_context_invalid", "Invalid facet selection for this workspace")
	if selections == nil {
		selections = map[string]string{}
	}
	if len(selections) > 32 {
		return entities.UserStateScope{}, Scope{}, invalid
	}
	scope := Scope{Environment: s.environment, Facets: selections, Driver: "local", Source: "workspace", Labels: map[string]FacetLabel{}}
	var config driverConfig
	if err := json.Unmarshal(workspace.Configuration, &config); err != nil {
		return entities.UserStateScope{}, Scope{}, err
	}
	if config.UserStateDriver != "" {
		scope.Driver = config.UserStateDriver
	}

	facets, err := s.facets.ListFacets(ctx, workspace.ID, false)
	if err != nil {
		return entities.UserStateScope{}, Scope{}, err
	}
	active := true
	for _, facet := range facets {
		if !facet.Active {
			continue
		}
		selection := selections[facet.Identity]
		if selection == "" {
			docs, err := s.facets.ListFacetDocuments(ctx, workspace.ID, facet.Identity, ports.DocumentFilter{Active: &active, Limit: 1})
			if err != nil {
				return entities.UserStateScope{}, Scope{}, err
			}
			if len(docs) != 0 {
				return entities.UserStateScope{}, Scope{}, invalid
			}
			continue
		}
		doc, err := s.facets.GetFacetDocument(ctx, workspace.ID, facet.Identity, selection, false)
		if errors.Is(err, ports.ErrNotFound) || (err == nil && !doc.Active) {
			return entities.UserStateScope{}, Scope{}, invalid
		}
		if err != nil {
			return entities.UserStateScope{}, Scope{}, err
		}
		scope.Labels[facet.Identity] = FacetLabel{Facet: facet.DisplayName, Document: doc.DisplayName}
		var contribution facetConfig
		if err := json.Unmarshal(doc.Data, &contribution); err != nil {
			return entities.UserStateScope{}, Scope{}, err
		}
		layer := contribution.Configuration
		switch {
		case layer.Mode == "replace":
			scope.Driver = layer.Value.UserStateDriver
			if scope.Driver == "" {
				scope.Driver = "local"
			}
			scope.Source = facet.Identity
		case layer.Patch.Driver.Operation == "set":
			scope.Driver = layer.Patch.Driver.Value
			scope.Source = facet.Identity
		}
	}
	if len(scope.Labels) != len(selections) {
		return entities.UserStateScope{}, Scope{}, invalid
	}
	encoded, err := json.Marshal(selections)
	key := entities.UserStateScope{WorkspaceID: workspace.ID, Environment: s.environment, Facets: string(encoded)}
	scope.Partition = partition(key)
	return key, scope, err
}
