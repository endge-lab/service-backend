package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func (r *EndgeRepository) StoreReleaseBuild(ctx context.Context, workspaceID, releaseID string, data []byte, metadata entities.ReleaseBuildMetadata) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE releases SET compiled_bundle=$3, build_metadata=$4 WHERE workspace_id=$1 AND id=$2 AND compiled_bundle IS NULL`, workspaceID, releaseID, data, encoded)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("release build is unavailable or already immutable")
	}
	return err
}

func (r *EndgeRepository) GetReleaseBuild(ctx context.Context, workspaceID, releaseID string) ([]byte, error) {
	var data []byte
	err := r.executor(ctx).QueryRow(ctx, `SELECT compiled_bundle FROM releases WHERE workspace_id=$1 AND id=$2 AND compiled_bundle IS NOT NULL`, workspaceID, releaseID).Scan(&data)
	return data, repositoryError(err)
}
