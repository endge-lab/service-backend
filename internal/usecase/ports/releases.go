package ports

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// ReleaseRepository задаёт порт хранения релизов для use case-слоя.
type ReleaseRepository interface {
	CreateRelease(context.Context, entities.Release, json.RawMessage) (*entities.Release, error)
	ListReleases(context.Context, string) ([]entities.Release, error)
	GetReleaseMetadata(context.Context, string, string) (*entities.Release, error)
	GetLatestReleaseMetadata(context.Context, string) (*entities.Release, error)
	GetReleaseArtifact(context.Context, string, string) (*entities.ReleaseArtifact, error)
}

// PortableRepository задаёт порт хранения переносимых пакетов для use case-слоя.
type PortableRepository interface {
	ExportWorkspace(context.Context, string, *int64) (json.RawMessage, error)
	ExportLiveWorkspace(context.Context, string) (json.RawMessage, error)
}

// SnapshotRepository задаёт хранение временных планов и страховочных копий полного импорта.
type SnapshotRepository interface {
	LockWorkspaceSnapshot(context.Context, string) error
	CreateSnapshotImportPlan(context.Context, entities.SnapshotImportPlan) (*entities.SnapshotImportPlan, error)
	GetSnapshotImportPlan(context.Context, string, string, string) (*entities.SnapshotImportPlan, error)
	MarkSnapshotImportPlanApplied(context.Context, string) error
	CreateSnapshotBackup(context.Context, entities.SnapshotBackup) (*entities.SnapshotBackup, error)
	ListSnapshotBackups(context.Context, string, string, bool, int, int) ([]entities.SnapshotBackup, error)
	GetSnapshotBackup(context.Context, string, string, bool) (*entities.SnapshotBackup, error)
}
