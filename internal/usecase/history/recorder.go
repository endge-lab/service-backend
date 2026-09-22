// Package history владеет записью mutation batches и неизменяемых ревизий.
package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

const SnapshotVersion = 1

// batchContextKey задаёт закрытый тип ключа контекста для пакета изменений.
type batchContextKey struct{}

// Recorder записывает пакеты изменений и ревизии сущностей.
type Recorder struct{ repository ports.RevisionRepository }

// NewRecorder создаёт сервис записи истории изменений.
func NewRecorder(repository ports.RevisionRepository) *Recorder {
	return &Recorder{repository: repository}
}

// BeginBatch создаёт общий mutation batch для атомарного действия над несколькими документами.
func (r *Recorder) BeginBatch(ctx context.Context, workspaceID *string, operation, actorID string) (context.Context, error) {
	batchID, err := r.repository.CreateMutationBatch(ctx, workspaceID, operation, actorID)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, batchContextKey{}, batchID), nil
}

// RecordDocument сохраняет полный снимок документа и увеличивает sequence рабочего пространства.
func (r *Recorder) RecordDocument(ctx context.Context, document entities.Document, operation string, restoredFrom *string) (*entities.Revision, error) {
	sequence, err := r.repository.NextWorkspaceSequence(ctx, document.WorkspaceID)
	if err != nil {
		return nil, err
	}
	batchID, err := r.batch(ctx, &document.WorkspaceID, operation, document.UpdatedBy.ID)
	if err != nil {
		return nil, err
	}
	parent, err := r.parent(ctx, &document.WorkspaceID, document.Type, document.ID)
	if err != nil {
		return nil, err
	}
	snapshot := marshal(document)
	revision := entities.Revision{
		ID: uuid.NewString(), WorkspaceID: document.WorkspaceID,
		DocumentType: document.Type, DocumentID: document.ID, DocumentIdentity: document.Identity,
		RevisionNumber: document.Revision, WorkspaceSequence: &sequence, Operation: operation,
		ParentRevisionID: parent, RestoredFromRevisionID: restoredFrom, MutationBatchID: batchID,
		SnapshotVersion: SnapshotVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: document.UpdatedBy,
	}
	return r.repository.InsertRevision(ctx, revision)
}

// RecordWorkspace сохраняет ревизию конфигурации рабочего пространства.
func (r *Recorder) RecordWorkspace(ctx context.Context, workspace entities.Workspace, operation string) error {
	sequence, err := r.repository.NextWorkspaceSequence(ctx, workspace.ID)
	if err != nil {
		return err
	}
	batchID, err := r.batch(ctx, &workspace.ID, operation, workspace.UpdatedBy.ID)
	if err != nil {
		return err
	}
	parent, err := r.parent(ctx, &workspace.ID, "workspaces", workspace.ID)
	if err != nil {
		return err
	}
	snapshot := marshal(workspace)
	_, err = r.repository.InsertRevision(ctx, entities.Revision{
		ID: uuid.NewString(), WorkspaceID: workspace.ID, DocumentType: "workspaces", DocumentID: workspace.ID,
		DocumentIdentity: workspace.Identity, RevisionNumber: workspace.Revision, WorkspaceSequence: &sequence,
		Operation: operation, ParentRevisionID: parent, MutationBatchID: batchID, SnapshotVersion: SnapshotVersion,
		Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: workspace.UpdatedBy,
	})
	return err
}

// RecordIntegration сохраняет ревизию глобальной интеграции без workspace sequence.
func (r *Recorder) RecordIntegration(ctx context.Context, integration entities.Integration, operation string, restoredFrom *string) error {
	batchID, err := r.batch(ctx, nil, operation, integration.UpdatedBy.ID)
	if err != nil {
		return err
	}
	parent, err := r.parent(ctx, nil, "integrations", integration.ID)
	if err != nil {
		return err
	}
	snapshot := marshal(integration)
	_, err = r.repository.InsertRevision(ctx, entities.Revision{
		ID: uuid.NewString(), DocumentType: "integrations", DocumentID: integration.ID,
		DocumentIdentity: integration.Identity, RevisionNumber: integration.Revision, Operation: operation,
		ParentRevisionID: parent, RestoredFromRevisionID: restoredFrom, MutationBatchID: batchID,
		SnapshotVersion: SnapshotVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: integration.UpdatedBy,
	})
	return err
}

// batch получает или создаёт пакет изменений для операции.
func (r *Recorder) batch(ctx context.Context, workspaceID *string, operation, actorID string) (string, error) {
	if batchID, ok := ctx.Value(batchContextKey{}).(string); ok && batchID != "" {
		return batchID, nil
	}
	return r.repository.CreateMutationBatch(ctx, workspaceID, operation, actorID)
}

// parent находит родительскую ревизию изменяемой сущности.
func (r *Recorder) parent(ctx context.Context, workspaceID *string, documentType, documentID string) (*string, error) {
	latest, err := r.repository.LatestRevision(ctx, workspaceID, documentType, documentID)
	if err == nil {
		return &latest.ID, nil
	}
	if errors.Is(err, ports.ErrNotFound) {
		return nil, nil
	}
	return nil, err
}

// marshal сериализует снимок сущности для истории изменений.
func marshal(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }

// checksum вычисляет SHA-256 контрольную сумму сериализованных данных.
func checksum(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
