package entities

// Collection names are part of the portable workspace contract. Keep them in
// one low-level package so repositories and use cases share the same values.
const (
	CollectionActions        = "actions"
	CollectionAuthProfiles   = "auth-profiles"
	CollectionComponents     = "components"
	CollectionCompositions   = "compositions"
	CollectionComputations   = "computations"
	CollectionConfigurations = "configurations"
	CollectionConverters     = "converters"
	CollectionDataViews      = "data-views"
	CollectionFilters        = "filters"
	CollectionFacetDocuments = "facet-documents"
	CollectionFacets         = "facets"
	CollectionFolders        = "folders"
	CollectionI18nBundles    = "i18n-bundles"
	CollectionMocks          = "mocks"
	CollectionNavigations    = "navigations"
	CollectionQueries        = "queries"
	CollectionStores         = "stores"
	CollectionStreams        = "streams"
	CollectionSimulations    = "simulations"
	CollectionStyles         = "styles"
	CollectionTypes          = "types"
	CollectionUpdates        = "updates"
	CollectionVocabs         = "vocabs"
)

// FacetCollections are persisted through explicit nested facet APIs. They are
// portable Domain collections, but are intentionally excluded from the generic
// document lifecycle because facet-document identity is scoped by its facet.
var FacetCollections = []string{CollectionFacets, CollectionFacetDocuments}

// DocumentCollections is the set of collections supported by the current
// portable workspace format.
var DocumentCollections = []string{
	CollectionFolders,
	CollectionTypes,
	CollectionQueries,
	CollectionDataViews,
	CollectionCompositions,
	CollectionStores,
	CollectionStreams,
	CollectionSimulations,
	CollectionUpdates,
	CollectionMocks,
	CollectionComponents,
	CollectionActions,
	CollectionFilters,
	CollectionConverters,
	CollectionComputations,
	CollectionVocabs,
	CollectionI18nBundles,
	CollectionAuthProfiles,
	CollectionNavigations,
	CollectionStyles,
	CollectionConfigurations,
}
