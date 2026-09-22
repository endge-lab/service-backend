package ports

import (
	"context"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// BridgeDelivery не блокирует application lock на сетевом I/O.
type BridgeDelivery interface {
	Send(string, entities.BridgeMessage) bool
	Close(string)
}

// BridgeAccessRepository перечитывает active user и отзыв browser session.
type BridgeAccessRepository interface {
	BridgeUser(context.Context, string, string) (entities.Actor, error)
}
