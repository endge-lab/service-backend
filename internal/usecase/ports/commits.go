package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// CommitRepository задаёт порт хранения коммитов для use case-слоя.
type CommitRepository interface {
	LatestCommit(context.Context, string) (*entities.Commit, error)
	ListCommits(context.Context, string) ([]entities.Commit, error)
	GetCommit(context.Context, string, string) (*entities.Commit, error)
	PendingRevisions(context.Context, string, int64) ([]entities.Revision, error)
	CreateCommit(context.Context, entities.Commit, []entities.CommitChange) (*entities.Commit, error)
	AttachRevisionsToCommit(context.Context, string, []string) error
	SquashPending(context.Context, string, int64, string, string) ([]entities.Revision, error)
}
