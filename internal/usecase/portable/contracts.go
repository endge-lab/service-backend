// Package portable содержит transport-independent contracts для logical
// export/import. Portable documents передают связи только через entity type и
// identity, поэтому не зависят от UUID конкретной базы или workspace.
package portable

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type ConflictPolicy string

const (
	ConflictPolicyFail      ConflictPolicy = "fail"
	ConflictPolicyOverwrite ConflictPolicy = "overwrite"
	ConflictPolicyRename    ConflictPolicy = "rename"
)

// EntityKey однозначно идентифицирует portable entity по её type и identity.
type EntityKey struct {
	EntityType string
	Identity   string
}

// PortableRelation описывает structured relation без foreign UUID. Path нужен
// adapter-у, чтобы сохранить target UUID в корректное поле canonical document.
type PortableRelation struct {
	Path       string `json:"path"`
	EntityType string `json:"entityType"`
	Identity   string `json:"identity"`
}

// PortableDocument содержит переносимое canonical представление одной entity.
// Canonical хранит source/JSON без подмены identity строк техническими UUID.
type PortableDocument struct {
	EntityType string             `json:"entityType"`
	Identity   string             `json:"identity"`
	Relations  []PortableRelation `json:"relations"`
	Canonical  json.RawMessage    `json:"canonical"`
}

// ResolvedRelation — внутренний результат import planner. UUID существует
// только между planner и adapter и никогда не попадает в PortableDocument.
type ResolvedRelation struct {
	Path       string
	EntityType string
	Identity   string
	TargetID   uuid.UUID
}

// ImportOptions задаёт policy конфликтов и atomic режим import.
// RenameIdentities содержит явно разрешённые переименования source key → target identity.
type ImportOptions struct {
	ConflictPolicy   ConflictPolicy
	RenameIdentities map[EntityKey]string
	Atomic           bool
}

// ImportError содержит ошибку одного portable document без технических storage details.
type ImportError struct {
	Document EntityKey `json:"document"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
}

// ImportResult перечисляет outcome logical import.
type ImportResult struct {
	Created []EntityKey   `json:"created"`
	Updated []EntityKey   `json:"updated"`
	Skipped []EntityKey   `json:"skipped"`
	Errors  []ImportError `json:"errors"`
}

// EntityPortableAdapter изолирует planner от entity-specific storage и
// canonical document formats. Реальные adapters добавляются задачами моделей.
type EntityPortableAdapter interface {
	EntityType() string
	Export(ctx context.Context, id uuid.UUID) (PortableDocument, error)
	FindByIdentity(ctx context.Context, identity string) (id uuid.UUID, found bool, err error)
	CreateBase(ctx context.Context, document PortableDocument) (uuid.UUID, error)
	OverwriteBase(ctx context.Context, id uuid.UUID, document PortableDocument) error
	ApplyRelations(ctx context.Context, id uuid.UUID, relations []ResolvedRelation) error
}
