package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type BuildProfileRepository interface {
	ListBuildProfiles(context.Context, string, string) ([]entities.BuildProfile, error)
	GetBuildProfile(context.Context, string, string) (*entities.BuildProfile, error)
	LockBuildProfileNames(context.Context, string) error
	NextBuildProfileNumber(context.Context, string) (int, error)
	InsertBuildProfile(context.Context, entities.BuildProfile) (*entities.BuildProfile, error)
	UpdateBuildProfile(context.Context, entities.BuildProfile, int) (*entities.BuildProfile, error)
	DeleteBuildProfile(context.Context, string, string, int) error
}
