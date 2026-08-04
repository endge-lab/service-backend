package commits

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// Plan описывает предварительный план операции и её последствия.
type Plan struct {
	BaseSequence  int64            `json:"baseSequence"`
	HeadSequence  int64            `json:"headSequence"`
	RevisionCount int              `json:"revisionCount"`
	DocumentCount int              `json:"documentCount"`
	Contributors  []entities.Actor `json:"contributors"`
	Shared        bool             `json:"shared"`
}

// Plan строит предварительный план операции без изменения состояния.
func (s *UseCase) Plan(ctx context.Context) (*Plan, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := s.repository.LatestCommit(ctx, scope.Workspace.ID)
	if err != nil {
		return nil, err
	}
	pending, err := s.repository.PendingRevisions(ctx, scope.Workspace.ID, latest.HeadSequence)
	if err != nil {
		return nil, err
	}
	documents, contributors := map[string]bool{}, map[string]entities.Actor{}
	for _, revision := range pending {
		documents[revision.DocumentType+":"+revision.DocumentID] = true
		contributors[revision.CreatedBy.ID] = revision.CreatedBy
	}
	actors := make([]entities.Actor, 0, len(contributors))
	for _, actor := range contributors {
		actors = append(actors, actor)
	}
	return &Plan{BaseSequence: latest.HeadSequence, HeadSequence: scope.Workspace.HeadSequence, RevisionCount: len(pending), DocumentCount: len(documents), Contributors: actors, Shared: len(contributors) > 1}, nil
}
