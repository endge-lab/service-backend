package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// RevisionRepository задаёт порт хранения ревизий для use case-слоя.
type RevisionRepository interface {
	NextWorkspaceSequence(context.Context, string) (int64, error)
	CreateMutationBatch(context.Context, *string, string, string) (string, error)
	LatestRevision(context.Context, *string, string, string) (*entities.Revision, error)
	InsertRevision(context.Context, entities.Revision) (*entities.Revision, error)
	ListRevisions(context.Context, string, string, string, RevisionListOptions) (RevisionPage, error)
	GetRevision(context.Context, string, string, string, string) (*entities.Revision, error)
}

// RevisionListOptions задаёт страницу истории документа.
// Limit 0 означает всю историю; IncludeSnapshot=false не читает снимки документа.
type RevisionListOptions struct {
	IncludeSnapshot bool
	Limit, Offset   int
}

// RevisionPage — страница ревизий от новых к старым и общее число ревизий документа.
type RevisionPage struct {
	Items []entities.Revision
	Total int
}
