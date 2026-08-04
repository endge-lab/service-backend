package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// IntegrationRepository задаёт порт хранения интеграций для use case-слоя.
type IntegrationRepository interface {
	ListIntegrations(context.Context, bool) ([]entities.Integration, error)
	GetIntegration(context.Context, string, bool) (*entities.Integration, error)
	InsertIntegration(context.Context, entities.Integration) (*entities.Integration, error)
	UpdateIntegration(context.Context, entities.Integration, int) (*entities.Integration, error)
}
