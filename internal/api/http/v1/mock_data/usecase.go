package mock_data

import (
	"context"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resource "github.com/endge-lab/service-backend/internal/usecase/mock_data"
	"time"
)

type UseCase interface {
	Capabilities(context.Context) (json.RawMessage, error)
	Generate(context.Context, json.RawMessage) (json.RawMessage, error)
	Create(context.Context, json.RawMessage) (entities.MockStream, error)
	Get(context.Context, string) (entities.MockStream, error)
	Update(context.Context, string, json.RawMessage) (entities.MockStream, error)
	KeepAlive(context.Context, string) (entities.MockStream, error)
	Stop(context.Context, string) error
	Subscribe(context.Context, string) (*resource.Subscription, error)
	ReserveBytes(int64) bool
	ReleaseBytes(int64)
	WriteTimeout() time.Duration
	RequestLimit() int
}

func BindUseCase(u *resource.UseCase) UseCase { return u }
