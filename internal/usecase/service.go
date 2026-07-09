package usecase

import (
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	LoadSession adapters.LoadSessionService
	Projects    adapters.ProjectService
	Folders     adapters.FolderService
}

type Params struct {
	fx.In

	Logger            *zap.Logger
	Tracer            trace.Tracer
	Metrics           *shared.UseCaseMetrics
	UserRepository    ports.UserRepository
	ProjectRepository ports.ProjectsRepository
	FolderRepository  ports.FoldersRepository
	TxManager         ports.TxManager
}

func NewService(params Params) *Service {
	factory := newServiceFactory(params)

	return &Service{
		LoadSession: factory.CreateLoadSessionUseCase(),
		Projects:    factory.CreateProjectsUseCase(),
		Folders:     factory.CreateFoldersUseCase(),
	}
}
