package postgres

import (
	"bytes"
	"context"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
)

const moduleColumns = "id::text, identity, name, description, revision, updated_by::text, deleted_at"
const moduleVersionColumns = "version, manifest, has_css, sha256, size, published_by::text, published_at"

func scanModule(row pgx.Row) (result entities.Module, err error) {
	err = row.Scan(&result.ID, &result.Identity, &result.Name, &result.Description, &result.Revision, &result.UpdatedBy, &result.DeletedAt)
	return result, repositoryError(err)
}

func scanModuleVersion(row pgx.Row, archive bool) (result entities.ModuleVersion, err error) {
	columns := []any{&result.Version, &result.Manifest, &result.HasCSS, &result.SHA256, &result.Size, &result.PublishedBy, &result.PublishedAt}
	if archive {
		columns = append(columns, &result.Archive)
	}
	err = row.Scan(columns...)
	return result, repositoryError(err)
}

func (r *EndgeRepository) ListModules(ctx context.Context, deleted bool) ([]entities.Module, error) {
	rows, err := r.executor(ctx).Query(ctx, "SELECT "+moduleColumns+" FROM modules WHERE (deleted_at IS NOT NULL)=$1 ORDER BY name, identity", deleted)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entities.Module, error) { return scanModule(row) })
}

func (r *EndgeRepository) GetModule(ctx context.Context, identity string) (entities.Module, error) {
	return scanModule(r.executor(ctx).QueryRow(ctx, "SELECT "+moduleColumns+" FROM modules WHERE identity=$1", identity))
}

func (r *EndgeRepository) CreateOrRestoreModule(ctx context.Context, value entities.Module) (entities.Module, error) {
	saved, err := scanModule(r.executor(ctx).QueryRow(ctx, `INSERT INTO modules (identity,name,description,updated_by)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (identity) DO UPDATE SET
			name=EXCLUDED.name, description=EXCLUDED.description, updated_by=EXCLUDED.updated_by,
			deleted_at=NULL, revision=modules.revision+1
		WHERE modules.deleted_at IS NOT NULL
		RETURNING `+moduleColumns,
		value.Identity, value.Name, value.Description, value.UpdatedBy))
	if errors.Is(err, ports.ErrNotFound) {
		return saved, ports.ErrModuleConflict
	}
	return saved, err
}

func (r *EndgeRepository) PatchModule(ctx context.Context, value entities.Module) (entities.Module, error) {
	saved, err := scanModule(r.executor(ctx).QueryRow(ctx, `UPDATE modules SET name=$1,description=$2,updated_by=$3,revision=revision+1
		WHERE id=$4 AND revision=$5 AND deleted_at IS NULL RETURNING `+moduleColumns, value.Name, value.Description, value.UpdatedBy, value.ID, value.Revision))
	if errors.Is(err, ports.ErrNotFound) {
		return saved, ports.ErrModuleConflict
	}
	return saved, err
}

func (r *EndgeRepository) SetModuleDeleted(ctx context.Context, id string, revision int64, actor string, deleted bool) (entities.Module, error) {
	saved, err := scanModule(r.executor(ctx).QueryRow(ctx, `UPDATE modules
		SET deleted_at=CASE WHEN $4 THEN NOW() ELSE NULL END,updated_by=$3,revision=revision+1
		WHERE id=$1 AND revision=$2 AND (deleted_at IS NOT NULL)<>$4 RETURNING `+moduleColumns, id, revision, actor, deleted))
	if errors.Is(err, ports.ErrNotFound) {
		return saved, ports.ErrModuleConflict
	}
	return saved, err
}

func (r *EndgeRepository) ListModuleVersions(ctx context.Context, moduleID string) ([]entities.ModuleVersion, error) {
	rows, err := r.executor(ctx).Query(ctx, "SELECT "+moduleVersionColumns+" FROM module_versions WHERE module_id=$1 ORDER BY published_at DESC, version", moduleID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entities.ModuleVersion, error) { return scanModuleVersion(row, false) })
}

func (r *EndgeRepository) GetModuleVersion(ctx context.Context, moduleID, version string, archive bool) (entities.ModuleVersion, error) {
	columns := moduleVersionColumns
	if archive {
		columns += ", archive"
	}
	return scanModuleVersion(r.executor(ctx).QueryRow(ctx, "SELECT "+columns+" FROM module_versions WHERE module_id=$1 AND version=$2", moduleID, version), archive)
}

func (r *EndgeRepository) PublishModuleVersion(ctx context.Context, moduleID string, value entities.ModuleVersion) (entities.ModuleVersion, error) {
	// The caller's transaction holds this lock through insertion so archiving and publishing are ordered.
	var activeID string
	if err := r.executor(ctx).QueryRow(ctx, "SELECT id::text FROM modules WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", moduleID).Scan(&activeID); err != nil {
		return entities.ModuleVersion{}, repositoryError(err)
	}
	saved, err := scanModuleVersion(r.executor(ctx).QueryRow(ctx, `INSERT INTO module_versions
		(module_id,version,manifest,has_css,archive,sha256,size,published_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (module_id,version) DO NOTHING RETURNING `+moduleVersionColumns,
		moduleID, value.Version, value.Manifest, value.HasCSS, value.Archive, value.SHA256, value.Size, value.PublishedBy), false)
	if !errors.Is(err, ports.ErrNotFound) {
		return saved, err
	}
	saved, err = r.GetModuleVersion(ctx, moduleID, value.Version, true)
	if err != nil {
		return saved, err
	}
	if !bytes.Equal(saved.Archive, value.Archive) {
		return entities.ModuleVersion{}, ports.ErrModuleConflict
	}
	saved.Archive = nil
	return saved, nil
}
