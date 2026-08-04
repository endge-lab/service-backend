package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ProviderID    string
	Subject       string
	Issuer        string
	Username      string
	DisplayName   string
	Nonce         string
	Groups        []string
	PlatformAdmin bool
	ExpiresAt     time.Time
}

type Resolver interface {
	Resolve(context.Context, string) (Claims, error)
}

type resolver struct {
	config  config.IdentityConfig
	keyfunc keyfunc.Keyfunc
}

func NewResolver(cfg *config.Config) (Resolver, error) {
	value := &resolver{config: cfg.Identity}
	if cfg.Identity.Mode == "oidc" {
		keys, err := keyfunc.NewDefaultCtx(context.Background(), []string{cfg.Identity.JWKSURL})
		if err != nil {
			return nil, fmt.Errorf("initialize OIDC JWKS: %w", err)
		}
		value.keyfunc = keys
	}
	return value, nil
}

func (r *resolver) Resolve(ctx context.Context, raw string) (Claims, error) {
	if r.config.Mode == "dev" {
		return Claims{ProviderID: "dev", Subject: r.config.DevSubject, Issuer: "urn:endge:dev", Username: r.config.DevUsername, DisplayName: r.config.DevDisplayName, PlatformAdmin: r.config.DevPlatformAdmin}, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Claims{}, fmt.Errorf("bearer token is required")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, r.keyfunc.KeyfuncCtx(ctx),
		jwt.WithValidMethods(r.config.AllowedAlgorithms), jwt.WithIssuer(r.config.Issuer), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid {
		return Claims{}, fmt.Errorf("invalid bearer token")
	}
	if kid, _ := token.Header["kid"].(string); strings.TrimSpace(kid) == "" {
		return Claims{}, fmt.Errorf("token kid is required")
	}

	subject, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(subject) == "" {
		return Claims{}, fmt.Errorf("token subject is required")
	}
	issuer, err := claims.GetIssuer()
	if err != nil || issuer != r.config.Issuer {
		return Claims{}, fmt.Errorf("token issuer is invalid")
	}
	audience, err := claims.GetAudience()
	if err != nil || !intersects(audience, r.config.AllowedAudiences) {
		return Claims{}, fmt.Errorf("token audience is invalid")
	}
	expires, err := claims.GetExpirationTime()
	if err != nil || expires == nil {
		return Claims{}, fmt.Errorf("token expiration is required")
	}

	groups := claimStrings(claims[r.config.GroupsClaim])
	return Claims{
		ProviderID: r.config.ProviderID, Subject: subject, Issuer: issuer,
		Username: claimString(claims[r.config.UsernameClaim]), DisplayName: claimString(claims[r.config.DisplayNameClaim]),
		Nonce: claimString(claims["nonce"]), Groups: groups, ExpiresAt: expires.Time,
		PlatformAdmin: slices.Contains(r.config.PlatformAdminSubjects, subject) || intersects(groups, r.config.PlatformAdminGroups),
	}, nil
}

func claimString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func claimStrings(value any) []string {
	switch typed := value.(type) {
	case string:
		return strings.Fields(typed)
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := claimString(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func intersects(left, right []string) bool {
	for _, value := range left {
		if slices.Contains(right, value) {
			return true
		}
	}
	return false
}
