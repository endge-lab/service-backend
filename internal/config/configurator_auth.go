package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	kitconfig "github.com/endge-lab/service-kit-go/config"
)

// ConfiguratorAuthConfig owns the browser login flow and the server-side
// Configurator session. Provider tokens never leave the backend.
type ConfiguratorAuthConfig struct {
	Adapter                string
	AuthorizationURL       string
	TokenURL               string
	LogoutURL              string
	ClientID               string
	ClientSecret           string
	Scopes                 []string
	RedirectURL            string
	ReturnURL              string
	AllowedReturnOrigins   []string
	SessionCookieName      string
	SessionTTL             time.Duration
	TransactionTTL         time.Duration
	SessionCleanupInterval time.Duration
	CookieSecure           bool
	CookieSameSite         string
	CookieDomain           string
}

func loadConfiguratorAuthConfig(base *kitconfig.ServiceConfig, identity IdentityConfig) (ConfiguratorAuthConfig, error) {
	configuratorAuth := ConfiguratorAuthConfig{
		Adapter:                env("AUTH_LOGIN_ADAPTER", loginAdapterDefault(identity.Mode)),
		AuthorizationURL:       env("AUTH_AUTHORIZATION_URL", ""),
		TokenURL:               env("AUTH_TOKEN_URL", ""),
		LogoutURL:              env("AUTH_LOGOUT_URL", ""),
		ClientID:               env("AUTH_CLIENT_ID", ""),
		ClientSecret:           strings.TrimSpace(os.Getenv("AUTH_CLIENT_SECRET")),
		Scopes:                 csv(env("AUTH_SCOPES", "openid,profile,email")),
		RedirectURL:            env("AUTH_REDIRECT_URL", strings.TrimRight(base.App.PublicURL, "/")+"/auth/callback"),
		ReturnURL:              env("AUTH_RETURN_URL", strings.TrimRight(base.App.PublicURL, "/")),
		AllowedReturnOrigins:   csv(os.Getenv("AUTH_ALLOWED_RETURN_ORIGINS")),
		SessionCookieName:      env("AUTH_SESSION_COOKIE_NAME", "endge_configurator_session"),
		SessionTTL:             envDuration("AUTH_SESSION_TTL", 8*time.Hour),
		TransactionTTL:         envDuration("AUTH_TRANSACTION_TTL", 10*time.Minute),
		SessionCleanupInterval: envDuration("AUTH_SESSION_CLEANUP_INTERVAL", 15*time.Minute),
		CookieSecure:           envBool("AUTH_COOKIE_SECURE", base.App.IsProduction()),
		CookieSameSite:         strings.ToLower(env("AUTH_COOKIE_SAME_SITE", "lax")),
		CookieDomain:           strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
	}
	if err := configuratorAuth.Validate(base.App.IsProduction()); err != nil {
		return ConfiguratorAuthConfig{}, err
	}
	return configuratorAuth, nil
}

func (c ConfiguratorAuthConfig) Validate(production bool) error {
	if c.CookieSameSite != "lax" && c.CookieSameSite != "strict" && c.CookieSameSite != "none" {
		return fmt.Errorf("AUTH_COOKIE_SAME_SITE must be lax, strict or none")
	}
	if c.CookieSameSite == "none" && !c.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true when AUTH_COOKIE_SAME_SITE=none")
	}
	if err := validateAuthHTTPURL("AUTH_RETURN_URL", c.ReturnURL, production, false); err != nil {
		return err
	}
	for _, origin := range c.AllowedReturnOrigins {
		if err := validateAuthHTTPURL("AUTH_ALLOWED_RETURN_ORIGINS", origin, production, true); err != nil {
			return err
		}
	}
	if c.Adapter == "dev" {
		if production {
			return fmt.Errorf("AUTH_LOGIN_ADAPTER=dev is forbidden in production")
		}
		return nil
	}
	if c.Adapter != "oidc" {
		return fmt.Errorf("AUTH_LOGIN_ADAPTER must be oidc or dev")
	}
	if c.AuthorizationURL == "" || c.TokenURL == "" || c.ClientID == "" || c.RedirectURL == "" {
		return fmt.Errorf("Configurator OIDC login configuration is incomplete")
	}
	for key, value := range map[string]string{
		"AUTH_AUTHORIZATION_URL": c.AuthorizationURL,
		"AUTH_TOKEN_URL":         c.TokenURL,
		"AUTH_REDIRECT_URL":      c.RedirectURL,
	} {
		if err := validateAuthHTTPURL(key, value, production, false); err != nil {
			return err
		}
	}
	if c.SessionTTL <= 0 || c.TransactionTTL <= 0 || c.SessionCleanupInterval <= 0 {
		return fmt.Errorf("auth session durations must be positive")
	}
	if production && !c.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in production")
	}
	return nil
}

func loginAdapterDefault(identityMode string) string {
	if identityMode == "dev" {
		return "dev"
	}
	return "oidc"
}

func validateAuthHTTPURL(key, value string, production, originOnly bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%s must contain absolute HTTP URLs without user info", key)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", key)
	}
	if originOnly && ((parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "") {
		return fmt.Errorf("%s entries must contain only scheme, host and optional port", key)
	}
	if production && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", key)
	}
	return nil
}
