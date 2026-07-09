package postgres

import (
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func mapServiceUser(user sqlc.ServiceUser) *entities.User {
	return &entities.User{
		ID:          user.ID.String(),
		AuthUserID:  user.AuthUserID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func mapProject(project sqlc.Project) *entities.Project {
	return &entities.Project{
		ID:          project.ID,
		Identity:    project.Identity,
		DisplayName: project.DisplayName,
		Description: mapNullableTextToEntity(project.Description),
		Active:      project.Active,
		DeletedAt:   mapNullableTimeToEntity(project.DeletedAt),
		Meta:        mapJSONBToEntity(project.Meta),
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

func mapCreateProjectParams(project *entities.Project) sqlc.CreateProjectParams {
	if project == nil {
		return sqlc.CreateProjectParams{}
	}

	return sqlc.CreateProjectParams{
		Identity:    project.Identity,
		DisplayName: project.DisplayName,
		Description: mapNullableTextToSQLC(project.Description),
		Active:      project.Active,
		Meta:        mapJSONBToSQLC(project.Meta),
	}
}

func mapUpdateProjectParams(project *entities.Project) sqlc.UpdateProjectParams {
	if project == nil {
		return sqlc.UpdateProjectParams{}
	}

	return sqlc.UpdateProjectParams{
		ID:          project.ID,
		DisplayName: project.DisplayName,
		Description: mapNullableTextToSQLC(project.Description),
		Active:      project.Active,
		Meta:        mapJSONBToSQLC(project.Meta),
	}
}

func mapNullableTextToEntity(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	text := value.String
	return &text
}

func mapNullableTextToSQLC(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *value,
		Valid:  true,
	}
}

func mapNullableUUIDToEntity(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	id := uuid.UUID(value.Bytes)
	return &id
}

func mapNullableUUIDToSQLC(value *uuid.UUID) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}

	return pgtype.UUID{
		Bytes: *value,
		Valid: true,
	}
}

func mapNullableTimeToEntity(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	t := value.Time
	return &t
}

func mapJSONBToEntity(value []byte) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}

	var meta map[string]any
	if err := json.Unmarshal(value, &meta); err != nil || meta == nil {
		return map[string]any{}
	}

	return meta
}

func mapJSONBToSQLC(value map[string]any) []byte {
	if value == nil {
		return []byte("{}")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}

	return data
}
