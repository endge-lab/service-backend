package usecase

import (
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/services"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	LoadSession adapters.LoadSessionService
	CreateTodo  adapters.CreateTodoService
}

type Params struct {
	fx.In

	Logger         *zap.Logger
	Tracer         trace.Tracer
	Metrics        *UseCaseMetrics
	UserRepository ports.UserRepository
	TodoRepository ports.TodoRepository
	TxManager      ports.TxManager
	TodoFactory    services.TodoFactory
}

func NewService(params Params) *Service {
	factory := newServiceFactory(params)

	return &Service{
		LoadSession: factory.CreateLoadSessionUseCase(),
		CreateTodo:  factory.CreateCreateTodoUseCase(),
	}
}
