package config

import (
	"fmt"
	"os"
	"strings"

	kitconfig "github.com/endge-lab/service-kit-go/config"
)

// IdentityConfig belongs to this service because login policy is part of the
// Configurator backend, not a reusable service-kit concern.
type IdentityConfig struct {
	Mode                  string
	ProviderID            string
	Issuer                string
	JWKSURL               string
	AllowedAudiences      []string
	AllowedAlgorithms     []string
	UsernameClaim         string
	DisplayNameClaim      string
	GroupsClaim           string
	PlatformAdminSubjects []string
	PlatformAdminGroups   []string
	DevSubject            string
	DevUsername           string
	DevDisplayName        string
	DevPlatformAdmin      bool
}

func loadIdentityConfig(base *kitconfig.ServiceConfig) (IdentityConfig, error) {
	identity := IdentityConfig{
		Mode:                  env("AUTH_MODE", modeDefault(base.App.IsProduction())),
		ProviderID:            env("AUTH_PROVIDER_ID", "primary"),
		Issuer:                env("AUTH_ISSUER", base.Auth.Issuer),
		JWKSURL:               env("AUTH_JWKS_URL", legacyJWKSURL(base.Auth)),
		AllowedAudiences:      csv(env("AUTH_ALLOWED_AUDIENCES", base.Auth.AllowedAudiences)),
		AllowedAlgorithms:     csv(env("AUTH_ALLOWED_ALGORITHMS", "RS256")),
		UsernameClaim:         env("AUTH_USERNAME_CLAIM", "preferred_username"),
		DisplayNameClaim:      env("AUTH_DISPLAY_NAME_CLAIM", "name"),
		GroupsClaim:           env("AUTH_GROUPS_CLAIM", "groups"),
		PlatformAdminSubjects: csv(os.Getenv("AUTH_PLATFORM_ADMIN_SUBJECTS")),
		PlatformAdminGroups:   csv(os.Getenv("AUTH_PLATFORM_ADMIN_GROUPS")),
		DevSubject:            env("AUTH_DEV_SUBJECT", "developer"),
		DevUsername:           env("AUTH_DEV_USERNAME", "developer"),
		DevDisplayName:        env("AUTH_DEV_DISPLAY_NAME", "Endge Developer"),
		DevPlatformAdmin:      envBool("AUTH_DEV_PLATFORM_ADMIN", true),
	}
	if err := identity.Validate(base.App.IsProduction()); err != nil {
		return IdentityConfig{}, err
	}
	return identity, nil
}

func legacyJWKSURL(value kitconfig.ServiceAuthConfig) string {
	if strings.TrimSpace(value.ServiceURL) == "" {
		return ""
	}
	return value.JWKSURL()
}

func (c IdentityConfig) Validate(production bool) error {
	switch c.Mode {
	case "oidc":
		if c.ProviderID == "" || c.Issuer == "" || c.JWKSURL == "" || len(c.AllowedAudiences) == 0 || len(c.AllowedAlgorithms) == 0 {
			return fmt.Errorf("OIDC auth configuration is incomplete")
		}
		for _, algorithm := range c.AllowedAlgorithms {
			upper := strings.ToUpper(strings.TrimSpace(algorithm))
			if upper == "NONE" || strings.HasPrefix(upper, "HS") {
				return fmt.Errorf("AUTH_ALLOWED_ALGORITHMS contains forbidden algorithm %q", algorithm)
			}
		}
	case "dev":
		if production {
			return fmt.Errorf("AUTH_MODE=dev is forbidden in production")
		}
		if c.DevSubject == "" {
			return fmt.Errorf("AUTH_DEV_SUBJECT is required in dev mode")
		}
	default:
		return fmt.Errorf("AUTH_MODE must be oidc or dev")
	}
	return nil
}

func modeDefault(production bool) string {
	if production {
		return "oidc"
	}
	return "dev"
}
