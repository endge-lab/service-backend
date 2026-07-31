package tenants

import (
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

var collectionPatchKeys = map[string]struct{}{
	entities.EndgeConfigurationPatchKeyVars:          {},
	entities.EndgeConfigurationPatchKeyLocales:       {},
	entities.EndgeConfigurationPatchKeyThemes:        {},
	entities.EndgeConfigurationPatchKeySFCAdapterIDs: {},
}

var scalarPatchKeys = map[string]struct{}{
	entities.EndgeConfigurationPatchKeySSE:                        {},
	entities.EndgeConfigurationPatchKeyDefaultLocale:              {},
	entities.EndgeConfigurationPatchKeyFallbackLocale:             {},
	entities.EndgeConfigurationPatchKeyDefaultTheme:               {},
	entities.EndgeConfigurationPatchKeyDefaultAuthProfileIdentity: {},
	entities.EndgeConfigurationPatchKeyDefaultSFCAdapterID:        {},
}

var requiredScalarPatchKeys = map[string]struct{}{
	entities.EndgeConfigurationPatchKeyDefaultLocale:       {},
	entities.EndgeConfigurationPatchKeyFallbackLocale:      {},
	entities.EndgeConfigurationPatchKeyDefaultTheme:        {},
	entities.EndgeConfigurationPatchKeyDefaultSFCAdapterID: {},
}

func normalizeCreateInput(input *CreateTenantInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Code = strings.TrimSpace(input.Code)
	if input.Identity == "" || input.DisplayName == "" || input.Code == "" {
		return apperrors.InvalidInput("validation_error", "tenant identity, display name and code are required")
	}
	if input.FolderIdentity != nil {
		value := strings.TrimSpace(*input.FolderIdentity)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "tenant folder identity is required")
		}
		input.FolderIdentity = &value
	}
	if input.Configuration == nil {
		configuration := entities.DefaultEndgeConfigurationContribution()
		input.Configuration = &configuration
	}
	return validateConfigurationContribution(*input.Configuration)
}

func normalizeUpdateInput(input *UpdateTenantInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	if input.Identity == "" {
		return apperrors.InvalidInput("validation_error", "tenant identity is required")
	}
	if input.DisplayName != nil {
		value := strings.TrimSpace(*input.DisplayName)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "tenant display name is required")
		}
		input.DisplayName = &value
	}
	if input.Code != nil {
		value := strings.TrimSpace(*input.Code)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "tenant code is required")
		}
		input.Code = &value
	}
	if input.FolderIdentity.Set && input.FolderIdentity.Value != nil {
		value := strings.TrimSpace(*input.FolderIdentity.Value)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "tenant folder identity is required")
		}
		input.FolderIdentity.Value = &value
	}
	if input.ConfigurationSet {
		if input.Configuration == nil {
			return apperrors.InvalidInput("validation_error", "tenant configuration cannot be null")
		}
		if err := validateConfigurationContribution(*input.Configuration); err != nil {
			return err
		}
	}
	if input.DisplayName == nil && input.Code == nil && !input.Description.Set && !input.FolderIdentity.Set && !input.ConfigurationSet {
		return apperrors.InvalidInput("validation_error", "tenant update is empty")
	}
	return nil
}

func validateConfigurationContribution(contribution entities.EndgeConfigurationContribution) (err error) {
	defer func() {
		if err != nil {
			err = apperrors.InvalidInput("configuration_invalid", apperrors.SafeMessageOf(err))
		}
	}()
	if contribution.Patch == nil {
		return apperrors.InvalidInput("validation_error", "configuration patch is required")
	}
	switch contribution.Mode {
	case entities.EndgeConfigurationContributionModeInherit:
		if contribution.Value != nil {
			return apperrors.InvalidInput("validation_error", "inherit configuration cannot contain a replacement value")
		}
		for key, raw := range contribution.Patch {
			if _, isCollection := collectionPatchKeys[key]; isCollection {
				if err := validateCollectionPatch(key, raw); err != nil {
					return err
				}
				continue
			}
			if _, isScalar := scalarPatchKeys[key]; isScalar {
				if err := validateScalarPatch(key, raw); err != nil {
					return err
				}
				continue
			}
			return apperrors.InvalidInput("validation_error", "configuration patch key is not supported")
		}
		return nil
	case entities.EndgeConfigurationContributionModeReplace:
		if len(contribution.Patch) != 0 {
			return apperrors.InvalidInput("validation_error", "replace configuration cannot contain a patch")
		}
		if contribution.Value == nil {
			return apperrors.InvalidInput("validation_error", "replace configuration value is required")
		}
		return shared.ValidateEndgeConfiguration(*contribution.Value)
	default:
		return apperrors.InvalidInput("validation_error", "configuration mode is invalid")
	}
}

func validateCollectionPatch(_ string, raw json.RawMessage) error {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return apperrors.InvalidInput("validation_error", "collection patch is invalid")
	}
	entriesRaw, ok := body["entries"]
	if !ok {
		return apperrors.InvalidInput("validation_error", "collection patch entries are required")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(entriesRaw, &entries); err != nil || entries == nil {
		return apperrors.InvalidInput("validation_error", "collection patch entries are invalid")
	}
	for _, entry := range entries {
		var key string
		var operation entities.EndgeConfigurationPatchOperation
		if err := json.Unmarshal(entry["key"], &key); err != nil || strings.TrimSpace(key) == "" {
			return apperrors.InvalidInput("validation_error", "collection patch entry key is required")
		}
		if err := json.Unmarshal(entry["op"], &operation); err != nil || (operation != entities.EndgeConfigurationPatchOperationUpsert && operation != entities.EndgeConfigurationPatchOperationRemove) {
			return apperrors.InvalidInput("validation_error", "collection patch operation is invalid")
		}
		if operation == entities.EndgeConfigurationPatchOperationUpsert {
			if value, ok := entry["value"]; !ok || len(value) == 0 || string(value) == "null" {
				return apperrors.InvalidInput("validation_error", "collection upsert value is required")
			}
		}
	}
	return nil
}

func validateScalarPatch(key string, raw json.RawMessage) error {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return apperrors.InvalidInput("validation_error", "scalar patch is invalid")
	}
	var operation entities.EndgeConfigurationPatchOperation
	if err := json.Unmarshal(body["op"], &operation); err != nil || (operation != entities.EndgeConfigurationPatchOperationSet && operation != entities.EndgeConfigurationPatchOperationRemove) {
		return apperrors.InvalidInput("validation_error", "scalar patch operation is invalid")
	}
	if operation == entities.EndgeConfigurationPatchOperationRemove {
		if _, required := requiredScalarPatchKeys[key]; required {
			return apperrors.InvalidInput("validation_error", "required configuration scalar cannot be removed")
		}
		return nil
	}
	if value, ok := body["value"]; !ok || len(value) == 0 {
		return apperrors.InvalidInput("validation_error", "scalar set value is required")
	}
	return nil
}
