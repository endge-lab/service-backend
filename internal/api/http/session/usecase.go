package session

import (
	"context"

	usecasesession "github.com/endge-lab/service-backend/internal/usecase/session"
)

// UseCase is the inbound application contract required by the session HTTP adapter.
type UseCase interface {
	Execute(ctx context.Context, input usecasesession.LoadSessionInput) (*usecasesession.LoadSessionOutput, error)
}
