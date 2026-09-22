package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// ReleaseArtifactRepository читает только большой immutable JSON release из
// постоянного хранилища. Узкий порт позволяет кешу не зависеть от всех release
// операций.
type ReleaseArtifactRepository interface {
	GetReleaseArtifact(context.Context, string, string) (*entities.ReleaseArtifact, error)
}

// ReleaseArtifactOperation ограничивает low-cardinality label метрик cache.
// Operation не должна формироваться из release identity, UUID или user input.
type ReleaseArtifactOperation string

const (
	ReleaseArtifactOperationExport      ReleaseArtifactOperation = "export"
	ReleaseArtifactOperationRestorePlan ReleaseArtifactOperation = "restore_plan"
	ReleaseArtifactOperationRestore     ReleaseArtifactOperation = "restore"
)

// ReleaseArtifactReader возвращает artifact через единый bounded in-memory cache.
// Проверка доступа к workspace остаётся обязанностью вызывающего use case.
type ReleaseArtifactReader interface {
	Read(context.Context, ReleaseArtifactOperation, string, entities.Release) (*entities.ReleaseArtifact, error)
}
