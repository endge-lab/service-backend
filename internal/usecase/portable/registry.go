package portable

import (
	"strings"

	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// Registry хранит adapters по entity type. Он не зависит от HTTP, PostgreSQL
// или конкретных entity-моделей.
type Registry struct {
	adapters map[string]EntityPortableAdapter
}

func NewRegistry(adapters ...EntityPortableAdapter) (*Registry, error) {
	registry := &Registry{adapters: make(map[string]EntityPortableAdapter, len(adapters))}
	for _, adapter := range adapters {
		if err := registry.Register(adapter); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register добавляет adapter для одного entity type и запрещает неявную замену
// существующего adapter.
func (r *Registry) Register(adapter EntityPortableAdapter) error {
	if adapter == nil {
		return apperrors.InvalidInput("validation_error", "portable adapter is required")
	}
	entityType := strings.TrimSpace(adapter.EntityType())
	if entityType == "" {
		return apperrors.InvalidInput("validation_error", "portable adapter entity type is required")
	}
	if r.adapters == nil {
		r.adapters = make(map[string]EntityPortableAdapter)
	}
	if _, exists := r.adapters[entityType]; exists {
		return apperrors.Conflict("portable_adapter_conflict", "portable adapter is already registered")
	}
	r.adapters[entityType] = adapter
	return nil
}

func (r *Registry) Adapter(entityType string) (EntityPortableAdapter, bool) {
	adapter, ok := r.adapters[strings.TrimSpace(entityType)]
	return adapter, ok
}
