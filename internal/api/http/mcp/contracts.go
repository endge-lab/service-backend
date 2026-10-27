package mcp

import (
	"reflect"

	"github.com/endge-lab/service-backend/internal/api/http/v1/action"
	"github.com/endge-lab/service-backend/internal/api/http/v1/auth_profile"
	"github.com/endge-lab/service-backend/internal/api/http/v1/commit"
	"github.com/endge-lab/service-backend/internal/api/http/v1/component"
	"github.com/endge-lab/service-backend/internal/api/http/v1/composition"
	"github.com/endge-lab/service-backend/internal/api/http/v1/computation"
	"github.com/endge-lab/service-backend/internal/api/http/v1/configuration"
	"github.com/endge-lab/service-backend/internal/api/http/v1/converter"
	"github.com/endge-lab/service-backend/internal/api/http/v1/data_view"
	"github.com/endge-lab/service-backend/internal/api/http/v1/facet"
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
	"github.com/endge-lab/service-backend/internal/usecase/facets"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
)

// Contracts — transport DTO REST, которыми MCP-инструменты проверяют входные данные
// перед вызовом use case. Правила (обязательные поля, длины, форматы, запрет
// неизвестных полей) задаются только validate-тегами REST DTO и проверяются той же
// функцией shared.DecodeAndValidateJSON, поэтому ошибка совпадает с REST: validation_error
// с details.fields или invalid_body.
type Contracts struct {
	Documents map[string]shared.DocumentContract
	// FormatVersions — поле версии формата каждой коллекции, выведенное из её CreateRequest.
	FormatVersions map[string]FormatVersion
	Facets         FacetContracts
	Commit         shared.RequestContract
}

// FacetContracts разбирает тела фасетов и документов фасетов DTO REST /api/v1/facets
// и переводит их во вход use case тем же методом Input, что REST-обработчик.
type FacetContracts struct {
	Create         func([]byte) (facets.FacetCreateInput, error)
	Patch          func([]byte) (facets.FacetPatchInput, error)
	DocumentCreate func([]byte) (facets.FacetDocumentCreateInput, error)
	DocumentPatch  func([]byte) (facets.FacetDocumentPatchInput, error)
}

// documentBinding — DTO создания и изменения коллекции и поле версии её формата.
type documentBinding struct {
	contract shared.DocumentContract
	version  func(collection string) FormatVersion
}

func bindDocument[Create, Patch any](validator appvalidator.Validator) documentBinding {
	return documentBinding{
		contract: shared.NewDocumentContract[Create, Patch](validator),
		version: func(collection string) FormatVersion {
			return formatVersionOf(collection, reflect.TypeFor[Create]())
		},
	}
}

// decodeWith проверяет тело DTO Request и переводит его во вход use case.
func decodeWith[Request, Input any](validator appvalidator.Validator, convert func(Request) Input) func([]byte) (Input, error) {
	return func(body []byte) (Input, error) {
		request, err := shared.DecodeAndValidateJSON[Request](body, validator)
		if err != nil {
			var zero Input
			return zero, err
		}
		return convert(request), nil
	}
}

// BindContracts связывает коллекции документного API с DTO их REST-ресурсов.
func BindContracts(validator appvalidator.Validator) Contracts {
	bindings := map[string]documentBinding{
		entities.CollectionFolders:        bindDocument[folder.CreateRequest, folder.PatchRequest](validator),
		entities.CollectionTypes:          bindDocument[domain_type.CreateRequest, domain_type.PatchRequest](validator),
		entities.CollectionQueries:        bindDocument[query.CreateRequest, query.PatchRequest](validator),
		entities.CollectionDataViews:      bindDocument[data_view.CreateRequest, data_view.PatchRequest](validator),
		entities.CollectionCompositions:   bindDocument[composition.CreateRequest, composition.PatchRequest](validator),
		entities.CollectionStores:         bindDocument[store.CreateRequest, store.PatchRequest](validator),
		entities.CollectionStreams:        bindDocument[stream.CreateRequest, stream.PatchRequest](validator),
		entities.CollectionSimulations:    bindDocument[simulation.CreateRequest, simulation.PatchRequest](validator),
		entities.CollectionUpdates:        bindDocument[update.CreateRequest, update.PatchRequest](validator),
		entities.CollectionMocks:          bindDocument[mock.CreateRequest, mock.PatchRequest](validator),
		entities.CollectionComponents:     bindDocument[component.CreateRequest, component.PatchRequest](validator),
		entities.CollectionActions:        bindDocument[action.CreateRequest, action.PatchRequest](validator),
		entities.CollectionFilters:        bindDocument[filter.CreateRequest, filter.PatchRequest](validator),
		entities.CollectionConverters:     bindDocument[converter.CreateRequest, converter.PatchRequest](validator),
		entities.CollectionComputations:   bindDocument[computation.CreateRequest, computation.PatchRequest](validator),
		entities.CollectionVocabs:         bindDocument[vocab.CreateRequest, vocab.PatchRequest](validator),
		entities.CollectionI18nBundles:    bindDocument[i18n_bundle.CreateRequest, i18n_bundle.PatchRequest](validator),
		entities.CollectionAuthProfiles:   bindDocument[auth_profile.CreateRequest, auth_profile.PatchRequest](validator),
		entities.CollectionNavigations:    bindDocument[navigation.CreateRequest, navigation.PatchRequest](validator),
		entities.CollectionStyles:         bindDocument[style.CreateRequest, style.PatchRequest](validator),
		entities.CollectionConfigurations: bindDocument[configuration.CreateRequest, configuration.PatchRequest](validator),
	}
	contracts := Contracts{
		Documents:      make(map[string]shared.DocumentContract, len(bindings)),
		FormatVersions: make(map[string]FormatVersion, len(bindings)),
		Facets: FacetContracts{
			Create:         decodeWith(validator, facet.CreateRequest.Input),
			Patch:          decodeWith(validator, facet.PatchRequest.Input),
			DocumentCreate: decodeWith(validator, facet.DocumentCreateRequest.Input),
			DocumentPatch:  decodeWith(validator, facet.DocumentPatchRequest.Input),
		},
		Commit: shared.NewRequestContract[commit.CreateRequest](validator),
	}
	for collection, binding := range bindings {
		contracts.Documents[collection] = binding.contract
		contracts.FormatVersions[collection] = binding.version(collection)
	}
	return contracts
}
