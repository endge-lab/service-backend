package usecase

import (
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/components"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	LoadSession adapters.LoadSessionService
	Projects    adapters.ProjectService
	Folders     adapters.FolderService
	Components  *components.Component
	Converters  *converters.Converter
}

type Params struct {
	fx.In

	Logger              *zap.Logger
	Tracer              trace.Tracer
	Metrics             *shared.UseCaseMetrics
	UserRepository      ports.UserRepository
	ProjectRepository   ports.ProjectsRepository
	FolderRepository    ports.FoldersRepository
	ComponentRepository ports.ComponentsRepository
	ConverterRepository ports.ConvertersRepository
	TxManager           ports.TxManager
}

func NewService(params Params) *Service {
	factory := newServiceFactory(params)

	return &Service{
		LoadSession: factory.CreateLoadSessionUseCase(),
		Projects:    factory.CreateProjectsUseCase(),
		Folders:     factory.CreateFoldersUseCase(),
		Components:  factory.CreateComponentsUseCase(),
		Converters:  factory.CreateConvertersUseCase(),
	}
}
