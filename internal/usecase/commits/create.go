package commits

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// Create создаёт коммит из ожидающих ревизий рабочего пространства.
func (s *UseCase) Create(ctx context.Context, message, policy string, expected int64) (result *entities.Commit, err error) {
	current, scope, err := shared.AdminContext(ctx)
	if err != nil {
		return nil, err
	}
	if policy == "" {
		policy = "preserve"
	}
	if policy != "preserve" && policy != "squash" {
		return nil, domainerrors.InvalidInput("revision_policy_invalid", "revisionPolicy must be preserve or squash")
	}
	latest, err := s.repository.LatestCommit(ctx, scope.Workspace.ID)
	if err != nil {
		return nil, err
	}
	if expected != scope.Workspace.HeadSequence {
		return nil, domainerrors.Conflict("head_sequence_conflict", "Workspace changed after preview")
	}
	pending, err := s.repository.PendingRevisions(ctx, scope.Workspace.ID, latest.HeadSequence)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, domainerrors.Conflict("nothing_to_commit", "There are no pending revisions")
	}
	if policy == "squash" && !shared.CanAdmin(scope.Role) {
		for _, revision := range pending {
			if revision.CreatedBy.ID != current.User.ID {
				return nil, domainerrors.Forbidden("shared_squash_requires_admin", "Shared squash requires Workspace Admin")
			}
		}
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		selected := pending
		if policy == "squash" {
			batchID, txErr := s.revisions.CreateMutationBatch(txctx, &scope.Workspace.ID, "commit_squash", current.User.ID)
			if txErr != nil {
				return txErr
			}
			selected, txErr = s.repository.SquashPending(txctx, scope.Workspace.ID, latest.HeadSequence, current.User.ID, batchID)
			if txErr != nil {
				return txErr
			}
		}
		value := entities.Commit{ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, ParentCommitID: &latest.ID, BaseSequence: latest.HeadSequence, HeadSequence: scope.Workspace.HeadSequence, Message: strings.TrimSpace(message), RevisionPolicy: policy, Operation: "user", CreatedBy: entities.Actor{ID: current.User.ID}}
		created, txErr := s.repository.CreateCommit(txctx, value, changes(selected))
		if txErr != nil {
			return txErr
		}
		ids := make([]string, 0, len(selected))
		for _, revision := range selected {
			ids = append(ids, revision.ID)
		}
		if txErr = s.repository.AttachRevisionsToCommit(txctx, created.ID, ids); txErr != nil {
			return txErr
		}
		result = created
		return nil
	})
	return result, shared.MapConflict(err)
}

// changes сворачивает ревизии в список изменений коммита.
func changes(revisions []entities.Revision) []entities.CommitChange {
	groups, order := map[string][]entities.Revision{}, []string{}
	for _, revision := range revisions {
		key := revision.DocumentType + ":" + revision.DocumentID
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], revision)
	}
	result := make([]entities.CommitChange, 0, len(order))
	for _, key := range order {
		items := groups[key]
		first, last := items[0], items[len(items)-1]
		result = append(result, entities.CommitChange{DocumentType: last.DocumentType, DocumentID: last.DocumentID, BeforeRevisionID: first.ParentRevisionID, AfterRevisionID: &last.ID, Operation: last.Operation})
	}
	return result
}
