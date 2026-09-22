package workspace_state

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// Status returns the committed portable identity of the current workspace.
func (s *Coordinator) Status(ctx context.Context) (result *entities.DomainStatus, err error) {
	scope, err := access(ctx)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinReadTransaction(ctx, func(txctx context.Context) error {
		live, txErr := s.repository.GetWorkspace(txctx, scope.Workspace.Identity)
		if txErr != nil {
			return txErr
		}
		latest, txErr := s.repository.LatestCommit(txctx, live.ID)
		if txErr != nil {
			return txErr
		}
		pending, txErr := s.repository.PendingRevisions(txctx, live.ID, latest.HeadSequence)
		if txErr != nil {
			return txErr
		}
		lastVersion := latest.DomainVersion
		if !domainversion.IsCurrent(lastVersion) {
			raw, exportErr := s.repository.ExportWorkspace(txctx, live.ID, &latest.HeadSequence)
			if exportErr != nil {
				return exportErr
			}
			lastVersion, txErr = domainversion.ComputeRaw(raw)
			if txErr != nil {
				return txErr
			}
		}
		state := "dirty"
		currentVersion := ""
		if latest.HeadSequence == live.HeadSequence && len(pending) == 0 {
			state = "clean"
			currentVersion = lastVersion
		}
		result = &entities.DomainStatus{
			Workspace: live.Identity, State: state,
			DomainVersion: currentVersion, LastCommittedDomainVersion: lastVersion,
			CommitID: latest.ID, CommitMessage: latest.Message, CommittedAt: latest.CreatedAt,
			PendingRevisionCount: len(pending),
		}
		return nil
	})
	return result, err
}
