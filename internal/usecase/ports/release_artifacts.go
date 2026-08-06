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

// ReleaseArtifactReader возвращает artifact через единый bounded in-memory cache.
// Проверка доступа к workspace остаётся обязанностью вызывающего use case.
type ReleaseArtifactReader interface {
	Read(context.Context, string, string, entities.Release) (*entities.ReleaseArtifact, error)
}
