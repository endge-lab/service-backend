package workspace_state

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

const (
	TransferNone = "none"
	TransferOwn  = "own"
	TransferAll  = "all"
)

type ExportOptions struct {
	PrivateBuildProfiles string `json:"privateBuildProfiles"`
	PrivateAIConnections string `json:"privateAIConnections"`
	IncludePublicAI      bool   `json:"includePublicAI"`
}

func (s *Coordinator) ExportWithOptions(ctx context.Context, options ExportOptions) (json.RawMessage, error) {
	current, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	options.PrivateBuildProfiles, err = normalizeTransferMode(options.PrivateBuildProfiles)
	if err != nil {
		return nil, err
	}
	options.PrivateAIConnections, err = normalizeTransferMode(options.PrivateAIConnections)
	if err != nil {
		return nil, err
	}
	if options.PrivateBuildProfiles == TransferAll && !canAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required to export all private build profiles")
	}
	if (options.PrivateAIConnections == TransferAll || options.IncludePublicAI) && !current.PlatformAdmin {
		return nil, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required to export public or other users' AI connections")
	}

	var result json.RawMessage
	err = s.tx.WithinReadTransaction(ctx, func(txctx context.Context) error {
		raw, txErr := s.repository.ExportWorkspace(txctx, scope.Workspace.ID, nil)
		if txErr != nil {
			return txErr
		}
		var bundle entities.PortableBundle
		if txErr = json.Unmarshal(raw, &bundle); txErr != nil {
			return txErr
		}
		profiles, txErr := s.repository.ListBuildProfilesForTransfer(txctx, scope.Workspace.ID, current.User.ID, options.PrivateBuildProfiles)
		if txErr != nil {
			return txErr
		}
		bundle.BuildProfiles = make([]entities.PortableBuildProfile, 0, len(profiles))
		for _, profile := range profiles {
			bundle.BuildProfiles = append(bundle.BuildProfiles, entities.PortableBuildProfile{
				Identity: profile.Identity, DisplayName: profile.DisplayName, Visibility: profile.Visibility,
				OwnerLogin: profile.OwnerLogin, SettingsVersion: profile.SettingsVersion, Settings: profile.Settings,
			})
		}
		if options.PrivateAIConnections != TransferNone || options.IncludePublicAI {
			connections, listErr := s.repository.ListAIConnectionsForTransfer(txctx, current.User.ID, options.PrivateAIConnections, options.IncludePublicAI)
			if listErr != nil {
				return listErr
			}
			catalog := &entities.PortableAICatalog{Connections: make([]entities.PortableAIConnection, 0, len(connections))}
			for _, item := range connections {
				credential := ""
				if len(item.Credential) > 0 {
					credential, txErr = s.keyring.Decrypt(item.Credential, []byte("ai-provider-credential:"+item.Connection.ID))
					if txErr != nil {
						return fmt.Errorf("decrypt AI credential for export: %w", txErr)
					}
				}
				portable := entities.PortableAIConnection{
					Name: item.Connection.Name, Adapter: item.Connection.Adapter, BaseURL: item.Connection.BaseURL,
					Visibility: item.Connection.Visibility, OwnerLogin: item.OwnerLogin, Credential: credential,
					Enabled: item.Connection.Enabled, Models: make([]entities.PortableAIModel, 0, len(item.Models)),
				}
				for _, model := range item.Models {
					portable.Models = append(portable.Models, entities.PortableAIModel{
						ProviderModelID: model.ProviderModelID, DisplayName: model.DisplayName,
						Enabled: model.Enabled, Default: model.Default,
					})
				}
				catalog.Connections = append(catalog.Connections, portable)
			}
			bundle.AICatalog = catalog
		}
		result, txErr = json.Marshal(bundle)
		return txErr
	})
	return result, err
}

func normalizeTransferMode(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = TransferNone
	}
	if value != TransferNone && value != TransferOwn && value != TransferAll {
		return "", domainerrors.InvalidInput("export_option_invalid", "transfer mode must be none, own or all")
	}
	return value, nil
}
