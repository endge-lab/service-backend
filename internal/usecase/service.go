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
	Components  adapters.ComponentService
	Converters  adapters.ConverterService
	Queries     adapters.QueryService
	DataViews   adapters.DataViewService
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
	QueryRepository     ports.QueriesRepository
	DataViewRepository  ports.DataViewsRepository
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
		Queries:     factory.CreateQueriesUseCase(),
		DataViews:   factory.CreateDataViewsUseCase(),
	}
}
