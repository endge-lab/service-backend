package mappers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func NullableTextToEntity(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}
func NullableTextToSQLC(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
func NullableUUIDToEntity(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes)
	return &result
}
func NullableUUIDToSQLC(value *uuid.UUID) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *value, Valid: true}
}
func NullableTimeToEntity(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
func JSONBToEntity(value []byte) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(value, &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}
func JSONBToSQLC(value map[string]any) []byte {
	if value == nil {
		return []byte("{}")
	}
	result, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return result
}
