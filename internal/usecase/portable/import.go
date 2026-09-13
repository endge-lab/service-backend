package portable

import (
	"context"
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// PlanImport строит предварительный план операции без изменения состояния.
func (s *UseCase) PlanImport(ctx context.Context, bundle entities.PortableBundle) (*entities.ImportPlan, error) {
	return s.coordinator.PlanImport(ctx, bundle)
}

// PlanImportArtifact decrypts an encrypted artifact when needed and builds the regular server-side plan.
func (s *UseCase) PlanImportArtifact(ctx context.Context, artifact json.RawMessage, password string) (*entities.ImportPlan, error) {
	raw := artifact
	if artifactKind(artifact) == encryptedArtifactKind {
		var envelope EncryptedEnvelope
		if err := json.Unmarshal(artifact, &envelope); err != nil {
			return nil, err
		}
		var err error
		raw, err = decryptArtifact(envelope, password)
		if err != nil {
			return nil, err
		}
	}
	var bundle entities.PortableBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, err
	}
	return s.coordinator.PlanImport(ctx, bundle)
}

// Import импортирует переносимый пакет в рабочее пространство.
func (s *UseCase) Import(ctx context.Context, planID, confirmation, ifMatch string) (*entities.SnapshotImportResult, error) {
	return s.coordinator.Import(ctx, planID, confirmation, ifMatch)
}
