package ports

import (
	"context"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type MockSubscription interface {
	Recv() (*entities.MockEvent, error)
	Close()
}
type MockGeneratorGateway interface {
	ConnectedServiceInfoProvider
	Capabilities(context.Context) (json.RawMessage, error)
	Generate(context.Context, entities.MockOwner, json.RawMessage) (json.RawMessage, error)
	Create(context.Context, entities.MockOwner, json.RawMessage) (entities.MockStream, error)
	Get(context.Context, entities.MockOwner, string) (entities.MockStream, error)
	Update(context.Context, entities.MockOwner, string, json.RawMessage) (entities.MockStream, error)
	KeepAlive(context.Context, entities.MockOwner, string) (entities.MockStream, error)
	Stop(context.Context, entities.MockOwner, string) error
	Subscribe(context.Context, entities.MockOwner, string) (MockSubscription, error)
}
