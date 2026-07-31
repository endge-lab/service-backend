package mappers

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
)

// DomainDependencyUsage преобразует строку SQLC usage в domain-модель, не
// раскрывая технические детали PostgreSQL за пределы repository layer.
func DomainDependencyUsage(value sqlc.ListDomainDependencyUsagesRow) entities.DomainDependencyUsage {
	return entities.DomainDependencyUsage{
		OwnerType:         value.OwnerType,
		OwnerID:           value.OwnerID,
		OwnerIdentity:     value.OwnerIdentity,
		SourcePath:        value.SourcePath,
		VerificationState: entities.DomainDependencyVerificationState(value.VerificationState),
	}
}
