package usecase

import (
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	LoadSession adapters.LoadSessionService
	Projects    adapters.ProjectService
}

type Params struct {
	fx.In

	Logger            *zap.Logger
	Tracer            trace.Tracer
	Metrics           *shared.UseCaseMetrics
	UserRepository    ports.UserRepository
	ProjectRepository ports.ProjectsRepository
	TxManager         ports.TxManager
}

func NewService(params Params) *Service {
	factory := newServiceFactory(params)

	return &Service{
		LoadSession: factory.CreateLoadSessionUseCase(),
		Projects:    factory.CreateProjectsUseCase(),
	}
}
