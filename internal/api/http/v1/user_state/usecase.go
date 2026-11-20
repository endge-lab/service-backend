package user_state

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	state "github.com/endge-lab/service-backend/internal/usecase/user_state"
)

type UseCase interface {
	Context(ctx context.Context, facets map[string]string, groups []string, admin bool) (entities.CurrentActor, entities.UserStateScope, state.Scope, error)
	Scopes(ctx context.Context, groups []string) ([]state.Scope, error)
	Users(ctx context.Context, facets map[string]string, groups []string, search, after string) (state.UserPage, error)
	Read(ctx context.Context, facets map[string]string, groups []string, user string, admin bool) (state.Snapshot, error)
	Save(ctx context.Context, facets map[string]string, groups []string, user string, admin bool, batch state.Batch) (state.BatchResult, error)
	Reset(ctx context.Context, facets map[string]string, groups []string, user, confirmation, expectedPartition string) (int64, error)
}

func BindUseCase(useCase *state.UseCase) UseCase { return useCase }
