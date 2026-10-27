package mcp

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/action"
	"github.com/endge-lab/service-backend/internal/api/http/v1/auth_profile"
	"github.com/endge-lab/service-backend/internal/api/http/v1/commit"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/composition"
	"github.com/endge-lab/service-backend/internal/api/http/v1/computation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/configuration"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/filter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/folder"
	"github.com/endge-lab/service-backend/internal/api/http/v1/i18n_bundle"
	"github.com/endge-lab/service-backend/internal/api/http/v1/mock"
	"github.com/endge-lab/service-backend/internal/api/http/v1/navigation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/query"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/api/http/v1/simulation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/store"
	"github.com/endge-lab/service-backend/internal/api/http/v1/stream"
	"github.com/endge-lab/service-backend/internal/api/http/v1/style"
	domain_type "github.com/endge-lab/service-backend/internal/api/http/v1/type"
	"github.com/endge-lab/service-backend/internal/api/http/v1/update"
	"github.com/endge-lab/service-backend/internal/api/http/v1/vocab"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
)

// Contracts — transport DTO REST, которыми MCP-инструменты проверяют входные данные
// перед вызовом use case. Правила (обязательные поля, длины, форматы, запрет
// неизвестных полей) задаются только validate-тегами REST DTO и проверяются той же
// функцией shared.DecodeAndValidateJSON, поэтому ошибка совпадает с REST: validation_error
// с details.fields или invalid_body.
type Contracts struct {
	Documents map[string]shared.DocumentContract
	Commit    shared.RequestContract
}

// BindContracts связывает коллекции документного API с DTO их REST-ресурсов.
func BindContracts(validator appvalidator.Validator) Contracts {
	return Contracts{
		Documents: map[string]shared.DocumentContract{
			entities.CollectionFolders:        shared.NewDocumentContract[folder.CreateRequest, folder.PatchRequest](validator),
			entities.CollectionTypes:          shared.NewDocumentContract[domain_type.CreateRequest, domain_type.PatchRequest](validator),
			entities.CollectionQueries:        shared.NewDocumentContract[query.CreateRequest, query.PatchRequest](validator),
			entities.CollectionDataViews:      shared.NewDocumentContract[data_view.CreateRequest, data_view.PatchRequest](validator),
			entities.CollectionCompositions:   shared.NewDocumentContract[composition.CreateRequest, composition.PatchRequest](validator),
			entities.CollectionStores:         shared.NewDocumentContract[store.CreateRequest, store.PatchRequest](validator),
			entities.CollectionStreams:        shared.NewDocumentContract[stream.CreateRequest, stream.PatchRequest](validator),
			entities.CollectionSimulations:    shared.NewDocumentContract[simulation.CreateRequest, simulation.PatchRequest](validator),
			entities.CollectionUpdates:        shared.NewDocumentContract[update.CreateRequest, update.PatchRequest](validator),
			entities.CollectionMocks:          shared.NewDocumentContract[mock.CreateRequest, mock.PatchRequest](validator),
			entities.CollectionComponents:     shared.NewDocumentContract[component.CreateRequest, component.PatchRequest](validator),
			entities.CollectionActions:        shared.NewDocumentContract[action.CreateRequest, action.PatchRequest](validator),
			entities.CollectionFilters:        shared.NewDocumentContract[filter.CreateRequest, filter.PatchRequest](validator),
			entities.CollectionConverters:     shared.NewDocumentContract[converter.CreateRequest, converter.PatchRequest](validator),
			entities.CollectionComputations:   shared.NewDocumentContract[computation.CreateRequest, computation.PatchRequest](validator),
			entities.CollectionVocabs:         shared.NewDocumentContract[vocab.CreateRequest, vocab.PatchRequest](validator),
			entities.CollectionI18nBundles:    shared.NewDocumentContract[i18n_bundle.CreateRequest, i18n_bundle.PatchRequest](validator),
			entities.CollectionAuthProfiles:   shared.NewDocumentContract[auth_profile.CreateRequest, auth_profile.PatchRequest](validator),
			entities.CollectionNavigations:    shared.NewDocumentContract[navigation.CreateRequest, navigation.PatchRequest](validator),
			entities.CollectionStyles:         shared.NewDocumentContract[style.CreateRequest, style.PatchRequest](validator),
			entities.CollectionConfigurations: shared.NewDocumentContract[configuration.CreateRequest, configuration.PatchRequest](validator),
		},
		Commit: shared.NewRequestContract[commit.CreateRequest](validator),
	}
}
