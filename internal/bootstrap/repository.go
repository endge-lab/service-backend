package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/repo"
	"github.com/endge-lab/service-backend/internal/repo/ports"

	"go.uber.org/fx"
)

func RepositoryModules() fx.Option {
	return fx.Options(
		fx.Provide(
			repo.NewRepository,
			fx.Annotate(
				func(repository *repo.Repository) ports.UserRepository {
					return repository.User()
				},
				fx.As(new(ports.UserRepository)),
			),
			fx.Annotate(
				func(repository *repo.Repository) ports.TodoRepository {
					return repository.Todo()
				},
				fx.As(new(ports.TodoRepository)),
			),
			fx.Annotate(
				func(repository *repo.Repository) ports.TxManager {
					return repository.TxManager()
				},
				fx.As(new(ports.TxManager)),
			),
			fx.Annotate(
				func(repository *repo.Repository) ports.ProjectsRepository { return repository.Project() },
				fx.As(new(ports.ProjectsRepository)),
			),
		),
	)
}
