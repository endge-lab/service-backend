package auth

import (
	"context"
	"fmt"
)

// TokenSet is the provider response kept only inside the backend session.
type TokenSet struct {
	AccessToken   string
	RefreshToken  string
	IdentityToken string
	ExpiresIn     int64
}

// LoginAdapter isolates provider-specific browser login mechanics from the
// Configurator HTTP contract.
type LoginAdapter interface {
	ID() string
	LoginURL(state, codeChallenge, nonce string) (string, error)
	Exchange(context.Context, string, string) (TokenSet, error)
	Refresh(context.Context, string) (TokenSet, error)
	Logout(context.Context, string) error
}

type LoginAdapterRegistry struct {
	configured string
	adapters   map[string]LoginAdapter
}

func NewLoginAdapterRegistry(adapter *OIDCAdapter) *LoginAdapterRegistry {
	adapters := map[string]LoginAdapter{adapter.ID(): adapter}
	return &LoginAdapterRegistry{configured: adapter.config.Adapter, adapters: adapters}
}

func (r *LoginAdapterRegistry) Current() (LoginAdapter, error) {
	adapter, ok := r.adapters[r.configured]
	if !ok {
		return nil, fmt.Errorf("Configurator login adapter %q is not registered", r.configured)
	}
	return adapter, nil
}
