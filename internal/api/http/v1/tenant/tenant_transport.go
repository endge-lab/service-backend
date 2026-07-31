package tenant

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/tenants"
	"github.com/google/uuid"
)

type CreateTenantRequest struct {
	Identity       string                     `json:"identity" validate:"required,min=1,max=160" example:"tenant-default"`
	DisplayName    string                     `json:"displayName" validate:"required,min=1,max=255" example:"Default tenant"`
	Code           string                     `json:"code" validate:"required,min=1,max=160" example:"TENANT_DEFAULT"`
	Description    *string                    `json:"description" example:"Main business tenant"`
	FolderIdentity *string                    `json:"folderIdentity" validate:"omitempty,min=1,max=160" example:"root-tenants"`
	Configuration  *ConfigurationContribution `json:"configuration"`
}

// UpdateTenantRequest keeps field-presence flags private while exposing normal
// scalar OpenAPI fields. It lets the handler distinguish an omitted property
// from an explicit JSON null.
type UpdateTenantRequest struct {
	DisplayName    *string                    `json:"displayName" validate:"omitempty,min=1,max=255" example:"Renamed tenant"`
	Code           *string                    `json:"code" validate:"omitempty,min=1,max=160" example:"TENANT_RENAMED"`
	Description    *string                    `json:"description" example:"Updated business tenant"`
	FolderIdentity *string                    `json:"folderIdentity" example:"root-tenants"`
	Configuration  *ConfigurationContribution `json:"configuration"`

	descriptionSet    bool
	folderIdentitySet bool
	configurationSet  bool
}

func (r *UpdateTenantRequest) UnmarshalJSON(data []byte) error {
	type requestAlias UpdateTenantRequest
	var decoded requestAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*r = UpdateTenantRequest(decoded)
	_, r.descriptionSet = fields["description"]
	_, r.folderIdentitySet = fields["folderIdentity"]
	_, r.configurationSet = fields["configuration"]
	return nil
}

func (r UpdateTenantRequest) input(identity string) (tenants.UpdateTenantInput, error) {
	input := tenants.UpdateTenantInput{
		Identity:         identity,
		DisplayName:      r.DisplayName,
		Code:             r.Code,
		Description:      tenants.NullableString{Set: r.descriptionSet, Value: r.Description},
		FolderIdentity:   tenants.NullableString{Set: r.folderIdentitySet, Value: r.FolderIdentity},
		ConfigurationSet: r.configurationSet,
	}
	if r.Configuration != nil {
		configuration, err := r.Configuration.domain()
		if err != nil {
			return tenants.UpdateTenantInput{}, err
		}
		input.Configuration = configuration
	}
	return input, nil
}

// ConfigurationContribution is the HTTP/OpenAPI representation of a tenant
// contribution. Patch stays an object at transport level and is converted to
// map[string]json.RawMessage before reaching the usecase.
type ConfigurationContribution struct {
	Mode  entities.EndgeConfigurationContributionMode `json:"mode" enums:"inherit,replace" example:"inherit"`
	Patch map[string]any                              `json:"patch" swaggertype:"object"`
	Value *entities.EndgeConfiguration                `json:"value,omitempty"`
}

func (c ConfigurationContribution) domain() (*entities.EndgeConfigurationContribution, error) {
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var result entities.EndgeConfigurationContribution
	if err = json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func contributionResponse(value entities.EndgeConfigurationContribution) ConfigurationContribution {
	encoded, _ := json.Marshal(value)
	var result ConfigurationContribution
	_ = json.Unmarshal(encoded, &result)
	return result
}

type TenantResponse struct {
	ID             uuid.UUID                 `json:"id" example:"00000000-0000-4000-8000-000000000071"`
	Identity       string                    `json:"identity" example:"tenant-default"`
	DisplayName    string                    `json:"displayName" example:"Default tenant"`
	Code           string                    `json:"code" example:"TENANT_DEFAULT"`
	Description    *string                   `json:"description" example:"Main business tenant"`
	FolderIdentity *string                   `json:"folderIdentity" example:"root-tenants"`
	Configuration  ConfigurationContribution `json:"configuration"`
	CreatedAt      time.Time                 `json:"createdAt" example:"2026-07-25T10:00:00Z"`
	UpdatedAt      time.Time                 `json:"updatedAt" example:"2026-07-25T10:00:00Z"`
}

type TenantsListResponse struct {
	Items []*TenantResponse `json:"items"`
}

func response(view *tenants.TenantView) *TenantResponse {
	if view == nil || view.Tenant == nil {
		return nil
	}
	value := view.Tenant
	return &TenantResponse{
		ID: value.ID, Identity: value.Identity, DisplayName: value.DisplayName, Code: value.Code,
		Description: value.Description, FolderIdentity: view.FolderIdentity,
		Configuration: contributionResponse(redactedContribution(value.Configuration)), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func listResponse(views []*tenants.TenantView) TenantsListResponse {
	items := make([]*TenantResponse, 0, len(views))
	for _, view := range views {
		items = append(items, response(view))
	}
	return TenantsListResponse{Items: items}
}

func redactedContribution(value entities.EndgeConfigurationContribution) entities.EndgeConfigurationContribution {
	result := value
	if value.Value != nil {
		configuration := *value.Value
		if configuration.SSE != nil {
			sse := *configuration.SSE
			sse.ManualToken = nil
			configuration.SSE = &sse
		}
		result.Value = &configuration
	}
	if raw, ok := value.Patch[entities.EndgeConfigurationPatchKeySSE]; ok {
		var patch map[string]json.RawMessage
		if json.Unmarshal(raw, &patch) == nil {
			var rawValue map[string]any
			if valueRaw, exists := patch["value"]; exists && json.Unmarshal(valueRaw, &rawValue) == nil {
				delete(rawValue, "manualToken")
				if encoded, err := json.Marshal(rawValue); err == nil {
					patch["value"] = encoded
					if encodedPatch, err := json.Marshal(patch); err == nil {
						result.Patch = make(map[string]json.RawMessage, len(value.Patch))
						for key, patchValue := range value.Patch {
							result.Patch[key] = patchValue
						}
						result.Patch[entities.EndgeConfigurationPatchKeySSE] = encodedPatch
					}
				}
			}
		}
	}
	return result
}
