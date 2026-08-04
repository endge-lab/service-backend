package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
)

type OIDCAdapter struct {
	config config.ConfiguratorAuthConfig
	client *http.Client
}

func NewOIDCAdapter(cfg *config.Config) *OIDCAdapter {
	return &OIDCAdapter{
		config: cfg.ConfiguratorAuth,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *OIDCAdapter) ID() string { return "oidc" }

func (a *OIDCAdapter) LoginURL(state, codeChallenge, nonce string) (string, error) {
	endpoint, err := url.Parse(a.config.AuthorizationURL)
	if err != nil {
		return "", fmt.Errorf("parse OIDC authorization URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("response_type", "code")
	query.Set("client_id", a.config.ClientID)
	query.Set("redirect_uri", a.config.RedirectURL)
	query.Set("scope", strings.Join(a.config.Scopes, " "))
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func (a *OIDCAdapter) Exchange(ctx context.Context, code, verifier string) (TokenSet, error) {
	return a.token(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {a.config.RedirectURL},
	})
}

func (a *OIDCAdapter) Refresh(ctx context.Context, refreshToken string) (TokenSet, error) {
	return a.token(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (a *OIDCAdapter) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(a.config.LogoutURL) == "" || strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	form := url.Values{"client_id": {a.config.ClientID}, "refresh_token": {refreshToken}}
	if a.config.ClientSecret != "" {
		form.Set("client_secret", a.config.ClientSecret)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.LogoutURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create OIDC logout request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("perform OIDC logout: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("OIDC logout returned status %d", response.StatusCode)
	}
	return nil
}

func (a *OIDCAdapter) token(ctx context.Context, form url.Values) (TokenSet, error) {
	form.Set("client_id", a.config.ClientID)
	if a.config.ClientSecret != "" {
		form.Set("client_secret", a.config.ClientSecret)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenSet{}, fmt.Errorf("create OIDC token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return TokenSet{}, fmt.Errorf("perform OIDC token request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return TokenSet{}, fmt.Errorf("OIDC token endpoint returned status %d", response.StatusCode)
	}
	var payload struct {
		AccessToken   string `json:"access_token"`
		RefreshToken  string `json:"refresh_token"`
		IdentityToken string `json:"id_token"`
		ExpiresIn     int64  `json:"expires_in"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return TokenSet{}, fmt.Errorf("decode OIDC token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return TokenSet{}, fmt.Errorf("OIDC token response has no access token")
	}
	return TokenSet{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, IdentityToken: payload.IdentityToken, ExpiresIn: payload.ExpiresIn}, nil
}
