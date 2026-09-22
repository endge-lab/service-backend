package auth

import (
	"context"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/access"
)

type accessTokenResolver map[string]Claims

func (r accessTokenResolver) Resolve(_ context.Context, raw string) (Claims, error) {
	return r[raw], nil
}

func TestExternalAccessRequiresFreshVerifiedClaims(t *testing.T) {
	policy := access.NewPolicy("primary", "provider", "v1", nil)
	valid := Claims{ProviderID: "primary", Subject: "user", Issuer: "issuer", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Minute), TokenHash: "verified-hash", Attributes: map[string]any{}}
	for name, change := range map[string]func(*Claims){
		"missing iat":                 func(c *Claims) { c.IssuedAt = time.Time{} },
		"future iat":                  func(c *Claims) { c.IssuedAt = time.Now().Add(time.Hour) },
		"expired":                     func(c *Claims) { c.ExpiresAt = time.Now().Add(-time.Second) },
		"wrong provider":              func(c *Claims) { c.ProviderID = "other" },
		"missing verified attributes": func(c *Claims) { c.Attributes = nil },
		"missing fingerprint":         func(c *Claims) { c.TokenHash = "" },
	} {
		t.Run(name, func(t *testing.T) {
			claims := valid
			change(&claims)
			if _, err := MapExternalAccess(policy, claims); err == nil {
				t.Fatal("invalid external evidence accepted")
			}
		})
	}
	if snapshot, err := MapExternalAccess(policy, valid); err != nil || snapshot == nil || len(snapshot.Grants) != 0 {
		t.Fatalf("empty mapping must be valid: %v", err)
	}
	identity := valid
	identity.Nonce = "nonce"
	manager := &SessionManager{access: policy, resolver: accessTokenResolver{"access": valid}}
	if _, err := manager.withExternalAccess(t.Context(), identity, "access"); err != nil {
		t.Fatal(err)
	}
	identity.Subject = "other"
	if _, err := manager.withExternalAccess(t.Context(), identity, "access"); err == nil {
		t.Fatal("accepted access token for another login identity")
	}
}

func TestExternalSessionRefreshesWhenPolicyChanges(t *testing.T) {
	policy := access.NewPolicy("primary", "provider", "new", nil)
	claims := Claims{ProviderID: "primary", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour), TokenHash: "hash", Attributes: map[string]any{}}
	snapshot, err := MapExternalAccess(policy, claims)
	if err != nil {
		t.Fatal(err)
	}
	record := sessionRecord{IdentityRefreshAt: time.Now().Add(time.Hour), ExternalAccess: snapshot}
	manager := &SessionManager{access: policy}
	if manager.identityRefreshDue(record, time.Now()) {
		t.Fatal("fresh mapping unexpectedly requires refresh")
	}
	snapshot.ConfigVersion = "old"
	if !manager.identityRefreshDue(record, time.Now()) {
		t.Fatal("old policy mapping was reused")
	}
	record.ExternalAccess = nil
	if !manager.identityRefreshDue(record, time.Now()) {
		t.Fatal("session created in local mode was reused")
	}
}
