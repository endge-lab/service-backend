package tenants

import "github.com/endge-lab/service-backend/internal/domain/entities"

type CreateTenantInput struct {
	Identity       string
	DisplayName    string
	Code           string
	Description    *string
	FolderIdentity *string
	Configuration  *entities.EndgeConfigurationContribution
}

// NullableString preserves the difference between an omitted PATCH field and
// an explicit JSON null after the HTTP layer has decoded a request.
type NullableString struct {
	Set   bool
	Value *string
}

type UpdateTenantInput struct {
	Identity         string
	DisplayName      *string
	Code             *string
	Description      NullableString
	FolderIdentity   NullableString
	ConfigurationSet bool
	Configuration    *entities.EndgeConfigurationContribution
}

type ListTenantsInput struct {
	FolderIdentity *string
}

// TenantView is the application result consumed by transports. FolderIdentity
// is resolved from the persisted FolderID within the request workspace, so an
// HTTP adapter never needs direct repository access.
type TenantView struct {
	Tenant         *entities.RTenant
	FolderIdentity *string
}
