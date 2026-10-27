package mcp

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/actions"
	"github.com/endge-lab/service-backend/internal/usecase/auth_profiles"
	"github.com/endge-lab/service-backend/internal/usecase/commits"
	"github.com/endge-lab/service-backend/internal/usecase/components"
	"github.com/endge-lab/service-backend/internal/usecase/compositions"
	"github.com/endge-lab/service-backend/internal/usecase/computations"
	"github.com/endge-lab/service-backend/internal/usecase/configurations"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/filters"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/endge-lab/service-backend/internal/usecase/i18n_bundles"
	"github.com/endge-lab/service-backend/internal/usecase/mocks"
	"github.com/endge-lab/service-backend/internal/usecase/navigations"
	"github.com/endge-lab/service-backend/internal/usecase/portable"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
	"github.com/endge-lab/service-backend/internal/usecase/revisions"
	"github.com/endge-lab/service-backend/internal/usecase/session"
	"github.com/endge-lab/service-backend/internal/usecase/simulations"
	"github.com/endge-lab/service-backend/internal/usecase/stores"
	"github.com/endge-lab/service-backend/internal/usecase/streams"
	"github.com/endge-lab/service-backend/internal/usecase/styles"
	"github.com/endge-lab/service-backend/internal/usecase/types"
	"github.com/endge-lab/service-backend/internal/usecase/updates"
	"github.com/endge-lab/service-backend/internal/usecase/vocabs"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
	"go.uber.org/fx"
)

// DocumentUseCase — CRUD-порт коллекции документов; тот же use case обслуживает REST.
type DocumentUseCase interface {
	List(context.Context, ports.DocumentFilter) ([]entities.Document, error)
	Get(context.Context, string, bool) (*entities.Document, error)
	Create(context.Context, documents.CreateInput) (*entities.Document, error)
	Patch(context.Context, string, documents.PatchInput, int) (*entities.Document, error)
	Delete(context.Context, string, int) (*entities.Document, error)
	Restore(context.Context, string, int) (*entities.Document, error)
}

// SessionUseCase возвращает пользователя и доступные ему workspace (для токена — только его workspace).
type SessionUseCase interface {
	Current(context.Context) (*session.Result, error)
}

// WorkspaceAuthorizer проверяет доступ к workspace так же, как RequireWorkspace в REST,
// и перечисляет workspace пользователя с ролями (для выбора набора инструментов JWT-сессии).
type WorkspaceAuthorizer interface {
	Authorize(context.Context, string) (entities.WorkspaceAccess, error)
	ListAccess(context.Context) ([]entities.WorkspaceAccess, error)
}

// DomainUseCase читает рабочий snapshot workspace.
type DomainUseCase interface {
	Live(context.Context) (json.RawMessage, error)
}

// RevisionUseCase читает историю ревизий документа.
type RevisionUseCase interface {
	List(context.Context, string, string, ports.RevisionListOptions) (ports.RevisionPage, error)
}

// CommitUseCase строит план и фиксирует ожидающие ревизии.
type CommitUseCase interface {
	Plan(context.Context) (*commits.Plan, error)
	Create(context.Context, string, string, int64) (*entities.Commit, error)
}

// UseCases — входные порты MCP-инструментов. Documents индексирован именем коллекции.
type UseCases struct {
	Session    SessionUseCase
	Workspaces WorkspaceAuthorizer
	Domain     DomainUseCase
	Revisions  RevisionUseCase
	Commits    CommitUseCase
	Documents  map[string]DocumentUseCase
}

// ConcreteUseCases — application use case, которые MCP вызывает напрямую, без HTTP-петли.
type ConcreteUseCases struct {
	fx.In

	Session        *session.UseCase
	Workspaces     *workspaces.UseCase
	Portable       *portable.UseCase
	Revisions      *revisions.UseCase
	Commits        *commits.UseCase
	Folders        *folders.UseCase
	Types          *types.UseCase
	Queries        *queries.UseCase
	DataViews      *data_views.UseCase
	Compositions   *compositions.UseCase
	Stores         *stores.UseCase
	Streams        *streams.UseCase
	Simulations    *simulations.UseCase
	Updates        *updates.UseCase
	Mocks          *mocks.UseCase
	Components     *components.UseCase
	Actions        *actions.UseCase
	Filters        *filters.UseCase
	Converters     *converters.UseCase
	Computations   *computations.UseCase
	Vocabs         *vocabs.UseCase
	I18nBundles    *i18n_bundles.UseCase
	AuthProfiles   *auth_profiles.UseCase
	Navigations    *navigations.UseCase
	Styles         *styles.UseCase
	Configurations *configurations.UseCase
}

// BindUseCases предоставляет concrete use case как порты MCP.
func BindUseCases(in ConcreteUseCases) UseCases {
	return UseCases{
		Session:    in.Session,
		Workspaces: in.Workspaces,
		Domain:     in.Portable,
		Revisions:  in.Revisions,
		Commits:    in.Commits,
		Documents: map[string]DocumentUseCase{
			entities.CollectionFolders:        in.Folders,
			entities.CollectionTypes:          in.Types,
			entities.CollectionQueries:        in.Queries,
			entities.CollectionDataViews:      in.DataViews,
			entities.CollectionCompositions:   in.Compositions,
			entities.CollectionStores:         in.Stores,
			entities.CollectionStreams:        in.Streams,
			entities.CollectionSimulations:    in.Simulations,
			entities.CollectionUpdates:        in.Updates,
			entities.CollectionMocks:          in.Mocks,
			entities.CollectionComponents:     in.Components,
			entities.CollectionActions:        in.Actions,
			entities.CollectionFilters:        in.Filters,
			entities.CollectionConverters:     in.Converters,
			entities.CollectionComputations:   in.Computations,
			entities.CollectionVocabs:         in.Vocabs,
			entities.CollectionI18nBundles:    in.I18nBundles,
			entities.CollectionAuthProfiles:   in.AuthProfiles,
			entities.CollectionNavigations:    in.Navigations,
			entities.CollectionStyles:         in.Styles,
			entities.CollectionConfigurations: in.Configurations,
		},
	}
}
