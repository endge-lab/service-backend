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
	CollectionEnvironments   = "environments"
	CollectionFilters        = "filters"
	CollectionFolders        = "folders"
	CollectionI18nBundles    = "i18n-bundles"
	CollectionMocks          = "mocks"
	CollectionNavigations    = "navigations"
	CollectionProjects       = "projects"
	CollectionQueries        = "queries"
	CollectionStores         = "stores"
	CollectionStreams        = "streams"
	CollectionSimulations    = "simulations"
	CollectionStyles         = "styles"
	CollectionTenants        = "tenants"
	CollectionTypes          = "types"
	CollectionUpdates        = "updates"
	CollectionVocabs         = "vocabs"
)

// DocumentCollections is the set of collections supported by the current
// portable workspace format.
var DocumentCollections = []string{
	CollectionProjects,
	CollectionTenants,
	CollectionEnvironments,
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
