package workspaces

import (
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
)

func normalizeCreateInput(input *CreateWorkspaceInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Identity == "" || input.DisplayName == "" {
		return apperrors.InvalidInput("validation_error", "workspace identity and display name are required")
	}
	return nil
}

func normalizeUpdateInput(input *UpdateWorkspaceInput) error {
	input.Identity = strings.TrimSpace(input.Identity)
	if input.Identity == "" {
		return apperrors.InvalidInput("validation_error", "workspace identity is required")
	}
	if input.DisplayName != nil {
		value := strings.TrimSpace(*input.DisplayName)
		if value == "" {
			return apperrors.InvalidInput("validation_error", "workspace display name is required")
		}
		input.DisplayName = &value
	}
	if input.DisplayName == nil && input.Configuration == nil {
		return apperrors.InvalidInput("validation_error", "workspace update is empty")
	}
	return nil
}

func validateConfiguration(c entities.EndgeConfiguration) error {
	if c.Vars == nil || c.Locales == nil || c.Themes == nil || c.SFCAdapterIDs == nil || len(c.Locales) == 0 || len(c.Themes) == 0 || len(c.SFCAdapterIDs) == 0 {
		return apperrors.InvalidInput("validation_error", "workspace configuration arrays are invalid")
	}
	vars := map[string]struct{}{}
	for _, v := range c.Vars {
		name := strings.TrimSpace(v.Name)
		if name == "" {
			return apperrors.InvalidInput("validation_error", "workspace variable name is required")
		}
		if _, exists := vars[name]; exists {
			return apperrors.InvalidInput("validation_error", "workspace variable names must be unique")
		}
		vars[name] = struct{}{}
	}
	locales := map[string]struct{}{}
	for _, v := range c.Locales {
		code := strings.TrimSpace(v.Code)
		if code == "" || (v.Direction != entities.LocaleDirectionLTR && v.Direction != entities.LocaleDirectionRTL) {
			return apperrors.InvalidInput("validation_error", "workspace locale is invalid")
		}
		if _, exists := locales[code]; exists {
			return apperrors.InvalidInput("validation_error", "workspace locale codes must be unique")
		}
		locales[code] = struct{}{}
	}
	if _, ok := locales[c.DefaultLocale]; !ok {
		return apperrors.InvalidInput("validation_error", "workspace default locale must exist")
	}
	if _, ok := locales[c.FallbackLocale]; !ok {
		return apperrors.InvalidInput("validation_error", "workspace fallback locale must exist")
	}
	themes := map[string]struct{}{}
	for _, v := range c.Themes {
		identity := strings.TrimSpace(v.Identity)
		if identity == "" {
			return apperrors.InvalidInput("validation_error", "workspace theme identity is required")
		}
		if _, exists := themes[identity]; exists {
			return apperrors.InvalidInput("validation_error", "workspace theme identities must be unique")
		}
		themes[identity] = struct{}{}
	}
	if _, ok := themes[c.DefaultTheme]; !ok {
		return apperrors.InvalidInput("validation_error", "workspace default theme must exist")
	}
	adapters := map[string]struct{}{}
	for _, v := range c.SFCAdapterIDs {
		v = strings.TrimSpace(v)
		if v == "" {
			return apperrors.InvalidInput("validation_error", "workspace sfc adapter id is required")
		}
		if _, exists := adapters[v]; exists {
			return apperrors.InvalidInput("validation_error", "workspace sfc adapter ids must be unique")
		}
		adapters[v] = struct{}{}
	}
	if _, ok := adapters[c.DefaultSFCAdapterID]; !ok {
		return apperrors.InvalidInput("validation_error", "workspace default sfc adapter must exist")
	}
	if c.SSE != nil {
		switch c.SSE.AuthMode {
		case entities.SSEAuthModeInherit, entities.SSEAuthModeProfile, entities.SSEAuthModeManual, entities.SSEAuthModeNone:
		default:
			return apperrors.InvalidInput("validation_error", "workspace sse auth mode is invalid")
		}
		if c.SSE.AuthMode == entities.SSEAuthModeProfile && (c.SSE.AuthProfileIdentity == nil || strings.TrimSpace(*c.SSE.AuthProfileIdentity) == "") {
			return apperrors.InvalidInput("validation_error", "workspace sse auth profile identity is required")
		}
	}
	return nil
}
