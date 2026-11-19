package postgres

import (
	"bytes"
	"context"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
)

const moduleColumns = "id::text, identity, name, description, revision, updated_by::text"
const moduleVersionColumns = "version, manifest, has_css, sha256, size, published_by::text, published_at"

func scanModule(row pgx.Row) (result entities.Module, err error) {
	err = row.Scan(&result.ID, &result.Identity, &result.Name, &result.Description, &result.Revision, &result.UpdatedBy)
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

func (r *EndgeRepository) ListModules(ctx context.Context) ([]entities.Module, error) {
	rows, err := r.executor(ctx).Query(ctx, "SELECT "+moduleColumns+" FROM modules ORDER BY name, identity")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entities.Module, error) { return scanModule(row) })
}

func (r *EndgeRepository) GetModule(ctx context.Context, identity string) (entities.Module, error) {
	return scanModule(r.executor(ctx).QueryRow(ctx, "SELECT "+moduleColumns+" FROM modules WHERE identity=$1", identity))
}

func (r *EndgeRepository) CreateModule(ctx context.Context, value entities.Module) (entities.Module, error) {
	saved, err := scanModule(r.executor(ctx).QueryRow(ctx, `INSERT INTO modules (identity,name,description,updated_by)
		VALUES ($1,$2,$3,$4) ON CONFLICT (identity) DO NOTHING RETURNING `+moduleColumns,
		value.Identity, value.Name, value.Description, value.UpdatedBy))
	if errors.Is(err, ports.ErrNotFound) {
		return saved, ports.ErrModuleConflict
	}
	return saved, err
}

func (r *EndgeRepository) PatchModule(ctx context.Context, value entities.Module) (entities.Module, error) {
	saved, err := scanModule(r.executor(ctx).QueryRow(ctx, `UPDATE modules SET name=$1,description=$2,updated_by=$3,revision=revision+1
		WHERE id=$4 AND revision=$5 RETURNING `+moduleColumns, value.Name, value.Description, value.UpdatedBy, value.ID, value.Revision))
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
