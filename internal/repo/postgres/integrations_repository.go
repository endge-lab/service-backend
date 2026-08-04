package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func integrationSelect() string {
	return `SELECT i.id::text,i.identity,i.display_name,i.description,i.version,i.managed_by,i.managed_by_id,i.meta,i.active,i.deleted_at,i.revision,` + actorScan("cu") + `,` + actorScan("uu") + `,i.created_at,i.updated_at FROM integrations i JOIN service_users cu ON cu.id=i.created_by JOIN service_users uu ON uu.id=i.updated_by`
}
func scanIntegration(row scanner) (*entities.Integration, error) {
	v := &entities.Integration{}
	var created, updated []byte
	if err := row.Scan(&v.ID, &v.Identity, &v.DisplayName, &v.Description, &v.Version, &v.ManagedBy, &v.ManagedByID, &v.Meta, &v.Active, &v.DeletedAt, &v.Revision, &created, &updated, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	_ = json.Unmarshal(created, &v.CreatedBy)
	_ = json.Unmarshal(updated, &v.UpdatedBy)
	return v, nil
}
func (r *EndgeRepository) ListIntegrations(ctx context.Context, includeDeleted bool) ([]entities.Integration, error) {
	query := integrationSelect()
	if !includeDeleted {
		query += ` WHERE i.deleted_at IS NULL`
	}
	query += ` ORDER BY i.identity`
	rows, err := r.executor(ctx).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Integration{}
	for rows.Next() {
		v, err := scanIntegration(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
func (r *EndgeRepository) GetIntegration(ctx context.Context, identity string, includeDeleted bool) (*entities.Integration, error) {
	query := integrationSelect() + ` WHERE i.identity=$1`
	if !includeDeleted {
		query += ` AND i.deleted_at IS NULL`
	}
	return scanIntegration(r.executor(ctx).QueryRow(ctx, query, identity))
}
func (r *EndgeRepository) InsertIntegration(ctx context.Context, v entities.Integration) (*entities.Integration, error) {
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO integrations(id,identity,display_name,description,version,managed_by,managed_by_id,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12)`, v.ID, v.Identity, v.DisplayName, v.Description, v.Version, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.CreatedBy.ID, v.Revision)
	if err != nil {
		return nil, err
	}
	return r.GetIntegration(ctx, v.Identity, true)
}
func (r *EndgeRepository) UpdateIntegration(ctx context.Context, v entities.Integration, expected int) (*entities.Integration, error) {
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE integrations SET identity=$1,display_name=$2,description=$3,version=$4,managed_by=$5,managed_by_id=$6,meta=$7,active=$8,deleted_at=$9,updated_by=$10,updated_at=NOW(),revision=revision+1 WHERE id=$11 AND revision=$12`, v.Identity, v.DisplayName, v.Description, v.Version, v.ManagedBy, v.ManagedByID, v.Meta, v.Active, v.DeletedAt, v.UpdatedBy.ID, v.ID, expected)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetIntegration(ctx, v.Identity, true)
}
