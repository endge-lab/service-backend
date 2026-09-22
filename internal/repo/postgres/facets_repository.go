package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5/pgconn"
)

func facetSelect() string {
	return `SELECT f.id::text,f.workspace_id::text,f.identity,f.display_name,f.icon,f.color,f.position,f.meta,f.active,f.deleted_at,f.revision,` + actorScan("cu") + `,` + actorScan("uu") + `,f.created_at,f.updated_at,
		(SELECT count(*) FROM facet_documents fd WHERE fd.workspace_id=f.workspace_id AND fd.facet_id=f.id AND fd.deleted_at IS NULL)
		FROM facets f JOIN service_users cu ON cu.id=f.created_by JOIN service_users uu ON uu.id=f.updated_by`
}

func scanFacet(row scanner) (*entities.Document, error) {
	value := &entities.Document{Type: entities.CollectionFacets, ManagedBy: "user"}
	var icon, color string
	var position, documentCount int
	var created, updated []byte
	if err := row.Scan(&value.ID, &value.WorkspaceID, &value.Identity, &value.DisplayName, &icon, &color, &position, &value.Meta, &value.Active, &value.DeletedAt, &value.Revision, &created, &updated, &value.CreatedAt, &value.UpdatedAt, &documentCount); err != nil {
		return nil, repositoryError(err)
	}
	value.Data = mustJSON(map[string]any{"icon": icon, "color": color, "position": position, "documentCount": documentCount})
	_ = json.Unmarshal(created, &value.CreatedBy)
	_ = json.Unmarshal(updated, &value.UpdatedBy)
	return value, nil
}

func (r *EndgeRepository) ListFacets(ctx context.Context, workspaceID string, includeDeleted bool) ([]entities.Document, error) {
	query := facetSelect() + ` WHERE f.workspace_id=$1`
	if !includeDeleted {
		query += ` AND f.deleted_at IS NULL`
	}
	query += ` ORDER BY CASE WHEN f.deleted_at IS NULL THEN 0 ELSE 1 END,f.position,f.identity`
	rows, err := r.executor(ctx).Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Document{}
	for rows.Next() {
		value, scanErr := scanFacet(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetFacet(ctx context.Context, workspaceID, identity string, includeDeleted bool) (*entities.Document, error) {
	query := facetSelect() + ` WHERE f.workspace_id=$1 AND f.identity=$2`
	if !includeDeleted {
		query += ` AND f.deleted_at IS NULL`
	}
	return scanFacet(r.executor(ctx).QueryRow(ctx, query, workspaceID, identity))
}

func (r *EndgeRepository) InsertFacet(ctx context.Context, value entities.Document) (*entities.Document, error) {
	var data map[string]any
	_ = json.Unmarshal(value.Data, &data)
	if _, err := r.executor(ctx).Exec(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, value.WorkspaceID); err != nil {
		return nil, repositoryError(err)
	}
	position := intValue(data["position"])
	if position < 0 {
		if err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(MAX(position),-1)+1 FROM facets WHERE workspace_id=$1 AND deleted_at IS NULL`, value.WorkspaceID).Scan(&position); err != nil {
			return nil, err
		}
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO facets(id,workspace_id,identity,display_name,icon,color,position,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12)`, value.ID, value.WorkspaceID, value.Identity, value.DisplayName, stringValue(data["icon"]), stringValue(data["color"]), position, value.Meta, value.Active, value.DeletedAt, value.CreatedBy.ID, value.Revision)
	if err != nil {
		return nil, err
	}
	return r.GetFacet(ctx, value.WorkspaceID, value.Identity, true)
}

func (r *EndgeRepository) UpdateFacet(ctx context.Context, value entities.Document, expected int) (*entities.Document, error) {
	var data map[string]any
	_ = json.Unmarshal(value.Data, &data)
	if _, err := r.executor(ctx).Exec(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, value.WorkspaceID); err != nil {
		return nil, repositoryError(err)
	}
	var currentIdentity string
	var currentDeletedAt any
	if err := r.executor(ctx).QueryRow(ctx, `SELECT identity,deleted_at FROM facets WHERE id=$1 AND workspace_id=$2 AND revision=$3 FOR UPDATE`, value.ID, value.WorkspaceID, expected).Scan(&currentIdentity, &currentDeletedAt); err != nil {
		return nil, repositoryError(err)
	}
	if currentIdentity != value.Identity || (currentDeletedAt == nil && value.DeletedAt != nil) {
		var count int
		if err := r.executor(ctx).QueryRow(ctx, `SELECT count(*) FROM facet_documents WHERE workspace_id=$1 AND facet_id=$2 AND deleted_at IS NULL`, value.WorkspaceID, value.ID).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("facet has active documents")
		}
	}
	position := intValue(data["position"])
	if position < 0 {
		if err := r.executor(ctx).QueryRow(ctx, `SELECT COALESCE(MAX(position),-1)+1 FROM facets WHERE workspace_id=$1 AND deleted_at IS NULL AND id<>$2`, value.WorkspaceID, value.ID).Scan(&position); err != nil {
			return nil, err
		}
	}
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE facets SET identity=$1,display_name=$2,icon=$3,color=$4,position=$5,meta=$6,active=$7,deleted_at=$8,updated_by=$9,updated_at=NOW(),revision=revision+1 WHERE id=$10 AND workspace_id=$11 AND revision=$12`, value.Identity, value.DisplayName, stringValue(data["icon"]), stringValue(data["color"]), position, value.Meta, value.Active, value.DeletedAt, value.UpdatedBy.ID, value.ID, value.WorkspaceID, expected)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetFacet(ctx, value.WorkspaceID, value.Identity, true)
}

func (r *EndgeRepository) ReorderFacets(ctx context.Context, workspaceID string, order []ports.FacetOrderItem, actorID string) ([]entities.Document, error) {
	if _, err := r.executor(ctx).Exec(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, workspaceID); err != nil {
		return nil, repositoryError(err)
	}
	rows, err := r.executor(ctx).Query(ctx, `SELECT id::text,identity,position,revision FROM facets WHERE workspace_id=$1 AND deleted_at IS NULL ORDER BY position,identity FOR UPDATE`, workspaceID)
	if err != nil {
		return nil, err
	}
	type currentFacet struct {
		id, identity       string
		position, revision int
	}
	current := []currentFacet{}
	for rows.Next() {
		var value currentFacet
		if err = rows.Scan(&value.id, &value.identity, &value.position, &value.revision); err != nil {
			rows.Close()
			return nil, err
		}
		current = append(current, value)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(current) != len(order) {
		return nil, fmt.Errorf("facet reorder set mismatch")
	}
	byIdentity := make(map[string]currentFacet, len(current))
	for _, value := range current {
		byIdentity[value.identity] = value
	}
	seen := map[string]bool{}
	for _, item := range order {
		value, ok := byIdentity[item.Identity]
		if !ok || seen[item.Identity] {
			return nil, fmt.Errorf("facet reorder set mismatch")
		}
		if value.revision != item.ExpectedRevision {
			return nil, fmt.Errorf("revision conflict")
		}
		seen[item.Identity] = true
	}
	if len(current) == 0 {
		return []entities.Document{}, nil
	}
	if _, err = r.executor(ctx).Exec(ctx, `UPDATE facets SET position=position+1000000 WHERE workspace_id=$1 AND deleted_at IS NULL`, workspaceID); err != nil {
		return nil, err
	}
	changed := []entities.Document{}
	for position, item := range order {
		value := byIdentity[item.Identity]
		var tag pgconn.CommandTag
		if value.position == position {
			tag, err = r.executor(ctx).Exec(ctx, `UPDATE facets SET position=$1 WHERE id=$2 AND workspace_id=$3`, position, value.id, workspaceID)
		} else {
			tag, err = r.executor(ctx).Exec(ctx, `UPDATE facets SET position=$1,updated_by=$2,updated_at=NOW(),revision=revision+1 WHERE id=$3 AND workspace_id=$4 AND revision=$5`, position, actorID, value.id, workspaceID, item.ExpectedRevision)
		}
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, fmt.Errorf("revision conflict")
		}
		if value.position != position {
			updated, getErr := r.GetFacet(ctx, workspaceID, item.Identity, true)
			if getErr != nil {
				return nil, getErr
			}
			changed = append(changed, *updated)
		}
	}
	return changed, nil
}

func (r *EndgeRepository) CountActiveFacetDocuments(ctx context.Context, workspaceID, facetIdentity string) (int, error) {
	var count int
	err := r.executor(ctx).QueryRow(ctx, `SELECT count(*) FROM facet_documents fd JOIN facets f ON f.id=fd.facet_id AND f.workspace_id=fd.workspace_id WHERE fd.workspace_id=$1 AND f.identity=$2 AND fd.deleted_at IS NULL`, workspaceID, facetIdentity).Scan(&count)
	return count, err
}

func facetDocumentSelect() string {
	return `SELECT d.id::text,d.workspace_id::text,d.identity,d.display_name,d.description,f.identity,d.configuration,d.meta,d.active,d.deleted_at,d.revision,` + actorScan("cu") + `,` + actorScan("uu") + `,d.created_at,d.updated_at
		FROM facet_documents d JOIN facets f ON f.id=d.facet_id AND f.workspace_id=d.workspace_id JOIN service_users cu ON cu.id=d.created_by JOIN service_users uu ON uu.id=d.updated_by`
}

func scanFacetDocument(row scanner) (*entities.Document, error) {
	value := &entities.Document{Type: entities.CollectionFacetDocuments, ManagedBy: "user"}
	var facetIdentity string
	var configuration, created, updated []byte
	if err := row.Scan(&value.ID, &value.WorkspaceID, &value.Identity, &value.DisplayName, &value.Description, &facetIdentity, &configuration, &value.Meta, &value.Active, &value.DeletedAt, &value.Revision, &created, &updated, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, repositoryError(err)
	}
	value.Data = mustJSON(map[string]any{"facetIdentity": facetIdentity, "configuration": json.RawMessage(configuration)})
	_ = json.Unmarshal(created, &value.CreatedBy)
	_ = json.Unmarshal(updated, &value.UpdatedBy)
	return value, nil
}

func (r *EndgeRepository) ListFacetDocuments(ctx context.Context, workspaceID, facetIdentity string, filter ports.DocumentFilter) ([]entities.Document, error) {
	args := []any{workspaceID, facetIdentity}
	where := []string{"d.workspace_id=$1", "f.identity=$2"}
	if !filter.IncludeDeleted {
		where = append(where, "d.deleted_at IS NULL")
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		where = append(where, fmt.Sprintf("d.active=$%d", len(args)))
	}
	query := facetDocumentSelect() + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY d.identity`
	if filter.Limit > 0 {
		args = append(args, filter.Limit, filter.Offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := r.executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Document{}
	for rows.Next() {
		value, scanErr := scanFacetDocument(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) listAllFacetDocuments(ctx context.Context, workspaceID string, includeDeleted bool) ([]entities.Document, error) {
	query := facetDocumentSelect() + ` WHERE d.workspace_id=$1`
	if !includeDeleted {
		query += ` AND d.deleted_at IS NULL`
	}
	query += ` ORDER BY f.position,f.identity,d.identity`
	rows, err := r.executor(ctx).Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.Document{}
	for rows.Next() {
		value, scanErr := scanFacetDocument(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

func (r *EndgeRepository) GetFacetDocument(ctx context.Context, workspaceID, facetIdentity, identity string, includeDeleted bool) (*entities.Document, error) {
	query := facetDocumentSelect() + ` WHERE d.workspace_id=$1 AND f.identity=$2 AND d.identity=$3`
	if !includeDeleted {
		query += ` AND d.deleted_at IS NULL`
	}
	return scanFacetDocument(r.executor(ctx).QueryRow(ctx, query, workspaceID, facetIdentity, identity))
}

func facetIdentityFromDocument(value entities.Document) string {
	var data map[string]any
	_ = json.Unmarshal(value.Data, &data)
	return stringValue(data["facetIdentity"])
}

func configurationFromFacetDocument(value entities.Document) json.RawMessage {
	var data map[string]json.RawMessage
	_ = json.Unmarshal(value.Data, &data)
	if raw := data["configuration"]; len(raw) > 0 {
		return raw
	}
	return json.RawMessage(`{"mode":"inherit","patch":{}}`)
}

func (r *EndgeRepository) InsertFacetDocument(ctx context.Context, value entities.Document) (*entities.Document, error) {
	facetIdentity := facetIdentityFromDocument(value)
	if _, err := r.executor(ctx).Exec(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, value.WorkspaceID); err != nil {
		return nil, repositoryError(err)
	}
	var facetID string
	parentQuery := `SELECT id::text FROM facets WHERE workspace_id=$1 AND identity=$2 AND deleted_at IS NULL`
	if value.DeletedAt != nil {
		// Portable facet tombstones may contain nested tombstones. Active
		// documents still require an active parent.
		parentQuery = `SELECT id::text FROM facets WHERE workspace_id=$1 AND identity=$2`
	}
	if err := r.executor(ctx).QueryRow(ctx, parentQuery, value.WorkspaceID, facetIdentity).Scan(&facetID); err != nil {
		return nil, repositoryError(err)
	}
	_, err := r.executor(ctx).Exec(ctx, `INSERT INTO facet_documents(id,workspace_id,facet_id,identity,display_name,description,configuration,meta,active,deleted_at,created_by,updated_by,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12)`, value.ID, value.WorkspaceID, facetID, value.Identity, value.DisplayName, value.Description, configurationFromFacetDocument(value), value.Meta, value.Active, value.DeletedAt, value.CreatedBy.ID, value.Revision)
	if err != nil {
		return nil, err
	}
	return r.GetFacetDocument(ctx, value.WorkspaceID, facetIdentity, value.Identity, true)
}

func (r *EndgeRepository) UpdateFacetDocument(ctx context.Context, value entities.Document, expected int) (*entities.Document, error) {
	facetIdentity := facetIdentityFromDocument(value)
	if _, err := r.executor(ctx).Exec(ctx, `SELECT id FROM workspaces WHERE id=$1 FOR UPDATE`, value.WorkspaceID); err != nil {
		return nil, repositoryError(err)
	}
	var facetID string
	parentQuery := `SELECT id::text FROM facets WHERE workspace_id=$1 AND identity=$2 AND deleted_at IS NULL`
	if value.DeletedAt != nil {
		parentQuery = `SELECT id::text FROM facets WHERE workspace_id=$1 AND identity=$2`
	}
	if err := r.executor(ctx).QueryRow(ctx, parentQuery, value.WorkspaceID, facetIdentity).Scan(&facetID); err != nil {
		return nil, repositoryError(err)
	}
	tag, err := r.executor(ctx).Exec(ctx, `UPDATE facet_documents SET facet_id=$1,identity=$2,display_name=$3,description=$4,configuration=$5,meta=$6,active=$7,deleted_at=$8,updated_by=$9,updated_at=NOW(),revision=revision+1 WHERE id=$10 AND workspace_id=$11 AND revision=$12`, facetID, value.Identity, value.DisplayName, value.Description, configurationFromFacetDocument(value), value.Meta, value.Active, value.DeletedAt, value.UpdatedBy.ID, value.ID, value.WorkspaceID, expected)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("revision conflict")
	}
	return r.GetFacetDocument(ctx, value.WorkspaceID, facetIdentity, value.Identity, true)
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return -1
	}
}

var _ ports.FacetRepository = (*EndgeRepository)(nil)
