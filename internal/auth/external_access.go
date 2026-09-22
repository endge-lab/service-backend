package auth

import (
	"fmt"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/access"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// MapExternalAccess accepts only claims returned by Resolver after JWT verification.
func MapExternalAccess(policy *access.Policy, claims Claims) (*entities.ExternalAccessSnapshot, error) {
	if !policy.External() {
		return nil, nil
	}
	if claims.ProviderID != policy.Provider() || claims.IssuedAt.IsZero() || claims.TokenHash == "" || claims.Attributes == nil {
		return nil, fmt.Errorf("external access requires verified access token with iat")
	}
	now := time.Now()
	if !claims.ExpiresAt.After(now) || claims.IssuedAt.After(now.Add(30*time.Second)) || !claims.ExpiresAt.After(claims.IssuedAt) {
		return nil, fmt.Errorf("external access token timestamps are invalid")
	}
	return &entities.ExternalAccessSnapshot{ConfigVersion: policy.Version(), TokenHash: claims.TokenHash,
		IssuedAt: claims.IssuedAt, ExpiresAt: claims.ExpiresAt, Grants: policy.Map(claims.Attributes)}, nil
}
