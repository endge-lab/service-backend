package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func scanBackendConnection(row scanner) (*entities.BackendConnection, error) {
	value := &entities.BackendConnection{}
	if err := row.Scan(&value.ID, &value.Name, &value.BaseURL, &value.CreatedBy, &value.CreatedAt); err != nil {
		return nil, repositoryError(err)
	}
	return value, nil
}

// ListBackendConnections возвращает глобальный каталог, отсортированный по названию и URL.
func (r *EndgeRepository) ListBackendConnections(ctx context.Context) ([]entities.BackendConnection, error) {
	rows, err := r.executor(ctx).Query(ctx, `SELECT id::text,name,base_url,created_by::text,created_at FROM backend_connections ORDER BY LOWER(name),base_url`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entities.BackendConnection{}
	for rows.Next() {
		value, scanErr := scanBackendConnection(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}

// InsertBackendConnection добавляет именованное подключение в каталог.
func (r *EndgeRepository) InsertBackendConnection(ctx context.Context, value entities.BackendConnection) (*entities.BackendConnection, error) {
	return scanBackendConnection(r.executor(ctx).QueryRow(ctx, `INSERT INTO backend_connections(id,name,base_url,created_by) VALUES($1,$2,$3,$4) RETURNING id::text,name,base_url,created_by::text,created_at`, value.ID, value.Name, value.BaseURL, value.CreatedBy))
}

// DeleteBackendConnection физически удаляет подключение.
func (r *EndgeRepository) DeleteBackendConnection(ctx context.Context, id string) error {
	tag, err := r.executor(ctx).Exec(ctx, `DELETE FROM backend_connections WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}
