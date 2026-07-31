package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EndgeConfigurationContributionMode string

const TenantRootFolderIdentity = "root-tenants"

const (
	EndgeConfigurationContributionModeInherit EndgeConfigurationContributionMode = "inherit"
	EndgeConfigurationContributionModeReplace EndgeConfigurationContributionMode = "replace"
)

type EndgeConfigurationPatchOperation string

const (
	EndgeConfigurationPatchOperationUpsert EndgeConfigurationPatchOperation = "upsert"
	EndgeConfigurationPatchOperationRemove EndgeConfigurationPatchOperation = "remove"
	EndgeConfigurationPatchOperationSet    EndgeConfigurationPatchOperation = "set"
)

const (
	EndgeConfigurationPatchKeyVars                       = "vars"
	EndgeConfigurationPatchKeyLocales                    = "locales"
	EndgeConfigurationPatchKeyThemes                     = "themes"
	EndgeConfigurationPatchKeySFCAdapterIDs              = "sfcAdapterIds"
	EndgeConfigurationPatchKeySSE                        = "sse"
	EndgeConfigurationPatchKeyDefaultLocale              = "defaultLocale"
	EndgeConfigurationPatchKeyFallbackLocale             = "fallbackLocale"
	EndgeConfigurationPatchKeyDefaultTheme               = "defaultTheme"
	EndgeConfigurationPatchKeyDefaultAuthProfileIdentity = "defaultAuthProfileIdentity"
	EndgeConfigurationPatchKeyDefaultSFCAdapterID        = "defaultSfcAdapterId"
)

// EndgeConfigurationContribution is one configuration layer in the cascade.
// Patch intentionally has no omitempty tag: an empty contribution is encoded
// as {"mode":"inherit","patch":{}}, not as an omitted patch field.
type EndgeConfigurationContribution struct {
	Mode  EndgeConfigurationContributionMode `json:"mode"`
	Patch map[string]json.RawMessage         `json:"patch"`
	Value *EndgeConfiguration                `json:"value,omitempty"`
}

// DefaultEndgeConfigurationContribution returns a contribution that inherits
// its entire upstream configuration without local changes.
func DefaultEndgeConfigurationContribution() EndgeConfigurationContribution {
	return EndgeConfigurationContribution{
		Mode:  EndgeConfigurationContributionModeInherit,
		Patch: map[string]json.RawMessage{},
	}
}

// RTenant is the final workspace-scoped configuration layer.
// It deliberately has no project, environment, or effective configuration:
// all three are determined only in a complete execution context.
type RTenant struct {
	ID            uuid.UUID                      `json:"id"`
	WorkspaceID   uuid.UUID                      `json:"workspace_id"`
	Identity      string                         `json:"identity"`
	DisplayName   string                         `json:"display_name"`
	Code          string                         `json:"code"`
	Description   *string                        `json:"description,omitempty"`
	FolderID      *uuid.UUID                     `json:"folder_id,omitempty"`
	Configuration EndgeConfigurationContribution `json:"configuration"`
	CreatedAt     time.Time                      `json:"created_at"`
	UpdatedAt     time.Time                      `json:"updated_at"`
}
