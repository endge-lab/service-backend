package entities

import (
	"time"

	"github.com/google/uuid"
)

type DomainDependencyVerificationState string

const (
	DomainDependencyVerificationStateVerified   DomainDependencyVerificationState = "verified"
	DomainDependencyVerificationStateUnverified DomainDependencyVerificationState = "unverified"
)

// DomainDependencyOwner идентифицирует canonical document, которому
// принадлежит derived dependency projection внутри одного workspace.
type DomainDependencyOwner struct {
	Type     string
	ID       uuid.UUID
	Identity string
}

// DomainDependencyReference описывает одну identity-ссылку из canonical source
// или authoring JSON. Технический UUID целевой сущности намеренно не хранится.
type DomainDependencyReference struct {
	Type       string
	Identity   string
	SourcePath string
}

// DomainDependency представляет одну сохранённую строку derived dependency.
type DomainDependency struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Owner       DomainDependencyOwner
	Reference   DomainDependencyReference
	CreatedAt   time.Time
}

// DomainDependencyState фиксирует, полностью ли canonical document owner
// проверен на dependencies.
type DomainDependencyState struct {
	WorkspaceID       uuid.UUID
	Owner             DomainDependencyOwner
	VerificationState DomainDependencyVerificationState
	VerificationError *string
	UpdatedAt         time.Time
}

// DomainDependencyUsage идентифицирует owner, который ссылается на заданную
// dependency identity. Эту структуру допустимо вернуть из read-only usages API.
type DomainDependencyUsage struct {
	OwnerType         string                            `json:"ownerType"`
	OwnerID           uuid.UUID                         `json:"ownerId"`
	OwnerIdentity     string                            `json:"ownerIdentity"`
	SourcePath        string                            `json:"sourcePath"`
	VerificationState DomainDependencyVerificationState `json:"verificationState"`
}

// DomainDependencyUsages содержит страницу usages для read-only API и
// delete guard.
type DomainDependencyUsages struct {
	Items  []DomainDependencyUsage
	Total  int64
	Limit  int
	Offset int
}
