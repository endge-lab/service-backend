package entities

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type workspaceContextKey struct{}

// WorkspaceScope is the workspace boundary resolved by the request layer.
type WorkspaceScope struct {
	ID       uuid.UUID
	Identity string
}

// WithWorkspace attaches the resolved workspace scope to an operation context.
func WithWorkspace(ctx context.Context, scope WorkspaceScope) context.Context {
	return context.WithValue(ctx, workspaceContextKey{}, scope)
}

// WorkspaceFromContext returns the resolved workspace scope of an operation.
func WorkspaceFromContext(ctx context.Context) (WorkspaceScope, bool) {
	scope, ok := ctx.Value(workspaceContextKey{}).(WorkspaceScope)
	return scope, ok && scope.ID != uuid.Nil
}

// WithWorkspaceID is kept for callers that only have an ID, such as existing tests.
func WithWorkspaceID(ctx context.Context, workspaceID uuid.UUID) context.Context {
	return WithWorkspace(ctx, WorkspaceScope{ID: workspaceID})
}

// WorkspaceIDFromContext returns the resolved workspace ID.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	scope, ok := WorkspaceFromContext(ctx)
	return scope.ID, ok
}

const (
	LocaleDirectionLTR = "ltr"
	LocaleDirectionRTL = "rtl"

	SSEAuthModeInherit = "inherit"
	SSEAuthModeProfile = "profile"
	SSEAuthModeManual  = "manual"
	SSEAuthModeNone    = "none"
)

// RWorkspace is the root organizational scope of the domain.
type RWorkspace struct {
	ID            uuid.UUID
	Identity      string
	DisplayName   string
	Configuration EndgeConfiguration
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EndgeConfiguration is the complete root configuration stored by a workspace.
type EndgeConfiguration struct {
	Vars                       []EndgeVariable        `json:"vars"`
	Locales                    []EndgeLocale          `json:"locales"`
	DefaultLocale              string                 `json:"defaultLocale" example:"ru"`
	FallbackLocale             string                 `json:"fallbackLocale" example:"ru"`
	Themes                     []EndgeTheme           `json:"themes"`
	DefaultTheme               string                 `json:"defaultTheme" example:"light"`
	DefaultAuthProfileIdentity *string                `json:"defaultAuthProfileIdentity" example:"default-auth"`
	SFCAdapterIDs              []string               `json:"sfcAdapterIds" example:"native-vue"`
	DefaultSFCAdapterID        string                 `json:"defaultSfcAdapterId" example:"native-vue"`
	SSE                        *EndgeSSEConfiguration `json:"sse,omitempty"`
}

// EndgeVariable declares a named workspace configuration variable.
// DefaultValue is optional because a variable can be declared before a
// concrete environment supplies its value.
type EndgeVariable struct {
	Name         string  `json:"name" validate:"required,min=1" example:"API_URL"`
	DefaultValue *string `json:"defaultValue,omitempty" example:"https://api.example.com"`
}

type EndgeLocale struct {
	Code        string `json:"code" example:"ru"`
	DisplayName string `json:"displayName" example:"Русский"`
	ShortLabel  string `json:"shortLabel" example:"RU"`
	Direction   string `json:"direction" enums:"ltr,rtl" example:"ltr"`
}

type EndgeTheme struct {
	Identity    string `json:"identity" example:"light"`
	DisplayName string `json:"displayName" example:"Светлая"`
}

type EndgeSSEConfiguration struct {
	URL                 string  `json:"url" example:"https://sse.example.com/events"`
	AuthMode            string  `json:"authMode" enums:"inherit,profile,manual,none" example:"inherit"`
	AuthProfileIdentity *string `json:"authProfileIdentity" example:"default-auth"`
	ManualToken         *string `json:"manualToken" example:"secret-token"`
}

// DefaultEndgeConfiguration returns the system configuration for new workspaces.
func DefaultEndgeConfiguration() EndgeConfiguration {
	return EndgeConfiguration{
		Vars: []EndgeVariable{},
		Locales: []EndgeLocale{
			{Code: "ru", DisplayName: "Русский", ShortLabel: "RU", Direction: LocaleDirectionLTR},
			{Code: "en", DisplayName: "English", ShortLabel: "EN", Direction: LocaleDirectionLTR},
		},
		DefaultLocale:  "ru",
		FallbackLocale: "ru",
		Themes: []EndgeTheme{
			{Identity: "light", DisplayName: "Светлая"},
			{Identity: "dark", DisplayName: "Тёмная"},
		},
		DefaultTheme:               "light",
		DefaultAuthProfileIdentity: nil,
		SFCAdapterIDs:              []string{"native-vue"},
		DefaultSFCAdapterID:        "native-vue",
	}
}

type CreateWorkspace struct {
	Identity      string
	DisplayName   string
	Configuration *EndgeConfiguration
}

type UpdateWorkspace struct {
	DisplayName   *string
	Configuration *EndgeConfiguration
}
