package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var rawURLEncoding = base64.RawURLEncoding

type SessionIdentity struct {
	Claims
	SessionID string
}

type LoginStart struct {
	Location     string
	BrowserNonce string
	ExpiresAt    time.Time
}

type SessionManager struct {
	config   config.ConfiguratorAuthConfig
	pool     *pgxpool.Pool
	registry *LoginAdapterRegistry
	resolver Resolver
	aead     cipher.AEAD
	loginURL string
}

func NewSessionManager(cfg *config.Config, pool *pgxpool.Pool, registry *LoginAdapterRegistry, resolver Resolver) (*SessionManager, error) {
	manager := &SessionManager{
		config: cfg.ConfiguratorAuth, pool: pool, registry: registry, resolver: resolver,
		loginURL: strings.TrimRight(cfg.App.PublicURL, "/") + "/auth/login",
	}
	if cfg.ConfiguratorAuth.Adapter == "dev" {
		return manager, nil
	}
	key, err := base64.StdEncoding.DecodeString(cfg.ConfiguratorAuth.SessionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode Configurator session encryption key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize Configurator session encryption: %w", err)
	}
	manager.aead, err = cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize Configurator session AEAD: %w", err)
	}
	return manager, nil
}

func (m *SessionManager) CookieName() string        { return m.config.SessionCookieName }
func (m *SessionManager) CookieSecure() bool        { return m.config.CookieSecure }
func (m *SessionManager) CookieDomain() string      { return m.config.CookieDomain }
func (m *SessionManager) SessionTTL() time.Duration { return m.config.SessionTTL }
func (m *SessionManager) LoginURL() string          { return m.loginURL }

func (m *SessionManager) Begin(ctx context.Context, requestedReturnURL string) (LoginStart, error) {
	if m.config.Adapter == "dev" {
		return LoginStart{Location: m.safeReturnURL(requestedReturnURL)}, nil
	}
	_, _ = m.pool.Exec(ctx, `DELETE FROM configurator_auth_transactions WHERE expires_at<=NOW()`)
	_, _ = m.pool.Exec(ctx, `DELETE FROM configurator_auth_sessions WHERE expires_at<=NOW() OR revoked_at<=NOW()-INTERVAL '1 day'`)
	adapter, err := m.registry.Current()
	if err != nil {
		return LoginStart{}, err
	}
	state, err := secureRandom(32)
	if err != nil {
		return LoginStart{}, err
	}
	verifier, err := secureRandom(48)
	if err != nil {
		return LoginStart{}, err
	}
	browserNonce, err := secureRandom(32)
	if err != nil {
		return LoginStart{}, err
	}
	oidcNonce, err := secureRandom(32)
	if err != nil {
		return LoginStart{}, err
	}
	verifierEncrypted, err := m.encrypt(verifier)
	if err != nil {
		return LoginStart{}, err
	}
	oidcNonceEncrypted, err := m.encrypt(oidcNonce)
	if err != nil {
		return LoginStart{}, err
	}
	expiresAt := time.Now().Add(m.config.TransactionTTL)
	_, err = m.pool.Exec(ctx, `
		INSERT INTO configurator_auth_transactions(state_hash,browser_nonce_hash,verifier_encrypted,oidc_nonce_encrypted,return_url,expires_at)
		VALUES($1,$2,$3,$4,$5,$6)`, hashToken(state), hashToken(browserNonce), verifierEncrypted, oidcNonceEncrypted,
		m.safeReturnURL(requestedReturnURL), expiresAt)
	if err != nil {
		return LoginStart{}, fmt.Errorf("store Configurator login transaction: %w", err)
	}
	challengeSum := sha256.Sum256([]byte(verifier))
	location, err := adapter.LoginURL(state, rawURLEncoding.EncodeToString(challengeSum[:]), oidcNonce)
	if err != nil {
		return LoginStart{}, err
	}
	return LoginStart{Location: location, BrowserNonce: browserNonce, ExpiresAt: expiresAt}, nil
}

func (m *SessionManager) Complete(ctx context.Context, state, code, browserNonce string) (string, string, time.Time, error) {
	if m.config.Adapter == "dev" {
		return "", m.config.ReturnURL, time.Time{}, fmt.Errorf("OIDC callback is unavailable in dev login mode")
	}
	if strings.TrimSpace(state) == "" || strings.TrimSpace(code) == "" || strings.TrimSpace(browserNonce) == "" {
		return "", "", time.Time{}, fmt.Errorf("OIDC callback state, code and browser binding are required")
	}
	var verifierEncrypted []byte
	var oidcNonceEncrypted []byte
	var returnURL string
	err := m.pool.QueryRow(ctx, `
		DELETE FROM configurator_auth_transactions
		WHERE state_hash=$1 AND browser_nonce_hash=$2 AND expires_at>NOW()
		RETURNING verifier_encrypted,oidc_nonce_encrypted,return_url`, hashToken(state), hashToken(browserNonce)).Scan(
		&verifierEncrypted, &oidcNonceEncrypted, &returnURL)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("consume Configurator login transaction: %w", err)
	}
	verifier, err := m.decrypt(verifierEncrypted)
	if err != nil {
		return "", "", time.Time{}, err
	}
	expectedOIDCNonce, err := m.decrypt(oidcNonceEncrypted)
	if err != nil {
		return "", "", time.Time{}, err
	}
	adapter, err := m.registry.Current()
	if err != nil {
		return "", "", time.Time{}, err
	}
	tokens, err := adapter.Exchange(ctx, code, verifier)
	if err != nil {
		return "", "", time.Time{}, err
	}
	claimsToken := tokens.IdentityToken
	if claimsToken == "" {
		claimsToken = tokens.AccessToken
	}
	claims, err := m.resolver.Resolve(ctx, claimsToken)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("validate OIDC callback identity: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedOIDCNonce)) != 1 {
		return "", "", time.Time{}, fmt.Errorf("OIDC identity nonce is invalid")
	}
	cookieToken, err := secureRandom(32)
	if err != nil {
		return "", "", time.Time{}, err
	}
	accessEncrypted, err := m.encrypt(tokens.AccessToken)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refreshEncrypted, err := m.encryptOptional(tokens.RefreshToken)
	if err != nil {
		return "", "", time.Time{}, err
	}
	groups, err := json.Marshal(claims.Groups)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("encode Configurator session groups: %w", err)
	}
	now := time.Now()
	accessExpiresAt := tokenExpiry(now, tokens.ExpiresIn, claims.ExpiresAt)
	sessionExpiresAt := now.Add(m.config.SessionTTL)
	_, err = m.pool.Exec(ctx, `
		INSERT INTO configurator_auth_sessions(
			token_hash,provider_id,subject,issuer,username,display_name,groups_json,platform_admin,
			access_token_encrypted,refresh_token_encrypted,access_expires_at,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		hashToken(cookieToken), claims.ProviderID, claims.Subject, claims.Issuer, claims.Username, claims.DisplayName,
		groups, claims.PlatformAdmin, accessEncrypted, nullableBytes(refreshEncrypted), accessExpiresAt, sessionExpiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("create Configurator session: %w", err)
	}
	return cookieToken, returnURL, sessionExpiresAt, nil
}

func (m *SessionManager) Resolve(ctx context.Context, cookieToken string) (SessionIdentity, error) {
	if strings.TrimSpace(cookieToken) == "" {
		return SessionIdentity{}, fmt.Errorf("Configurator session cookie is required")
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return SessionIdentity{}, fmt.Errorf("begin Configurator session resolution: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var record sessionRecord
	var groupsJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text,provider_id,subject,issuer,username,display_name,groups_json,platform_admin,
			access_token_encrypted,refresh_token_encrypted,access_expires_at,expires_at
		FROM configurator_auth_sessions
		WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW()
		FOR UPDATE`, hashToken(cookieToken)).Scan(
		&record.ID, &record.ProviderID, &record.Subject, &record.Issuer, &record.Username, &record.DisplayName,
		&groupsJSON, &record.PlatformAdmin, &record.AccessTokenEncrypted, &record.RefreshTokenEncrypted,
		&record.AccessExpiresAt, &record.ExpiresAt)
	if err != nil {
		return SessionIdentity{}, fmt.Errorf("Configurator session is invalid")
	}
	if err = json.Unmarshal(groupsJSON, &record.Groups); err != nil {
		return SessionIdentity{}, fmt.Errorf("decode Configurator session groups: %w", err)
	}
	if record.AccessExpiresAt.Before(time.Now().Add(30 * time.Second)) {
		identity, refreshErr := m.refresh(ctx, tx, cookieToken, record)
		if refreshErr != nil {
			_, _ = tx.Exec(ctx, `UPDATE configurator_auth_sessions SET revoked_at=NOW(),updated_at=NOW() WHERE token_hash=$1`, hashToken(cookieToken))
			_ = tx.Commit(ctx)
			return SessionIdentity{}, refreshErr
		}
		if err = tx.Commit(ctx); err != nil {
			return SessionIdentity{}, fmt.Errorf("commit refreshed Configurator session: %w", err)
		}
		return identity, nil
	}
	if err = tx.Commit(ctx); err != nil {
		return SessionIdentity{}, fmt.Errorf("commit Configurator session resolution: %w", err)
	}
	return record.identity(), nil
}

func (m *SessionManager) Revoke(ctx context.Context, cookieToken string) error {
	if strings.TrimSpace(cookieToken) == "" || m.config.Adapter == "dev" {
		return nil
	}
	var refreshEncrypted []byte
	err := m.pool.QueryRow(ctx, `
		UPDATE configurator_auth_sessions SET revoked_at=NOW(),updated_at=NOW()
		WHERE token_hash=$1 AND revoked_at IS NULL
		RETURNING refresh_token_encrypted`, hashToken(cookieToken)).Scan(&refreshEncrypted)
	if err != nil {
		return nil
	}
	refreshToken, err := m.decryptOptional(refreshEncrypted)
	if err != nil || refreshToken == "" {
		return err
	}
	adapter, err := m.registry.Current()
	if err != nil {
		return err
	}
	return adapter.Logout(ctx, refreshToken)
}

func (m *SessionManager) refresh(ctx context.Context, tx pgx.Tx, cookieToken string, record sessionRecord) (SessionIdentity, error) {
	refreshToken, err := m.decryptOptional(record.RefreshTokenEncrypted)
	if err != nil || refreshToken == "" {
		return SessionIdentity{}, fmt.Errorf("Configurator session has expired")
	}
	adapter, err := m.registry.Current()
	if err != nil {
		return SessionIdentity{}, err
	}
	tokens, err := adapter.Refresh(ctx, refreshToken)
	if err != nil {
		return SessionIdentity{}, fmt.Errorf("refresh Configurator session: %w", err)
	}
	claimsToken := tokens.IdentityToken
	if claimsToken == "" {
		claimsToken = tokens.AccessToken
	}
	claims, err := m.resolver.Resolve(ctx, claimsToken)
	if err != nil {
		return SessionIdentity{}, fmt.Errorf("validate refreshed Configurator identity: %w", err)
	}
	if claims.ProviderID != record.ProviderID || claims.Subject != record.Subject || claims.Issuer != record.Issuer {
		return SessionIdentity{}, fmt.Errorf("refreshed Configurator identity does not match the session")
	}
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = refreshToken
	}
	accessEncrypted, err := m.encrypt(tokens.AccessToken)
	if err != nil {
		return SessionIdentity{}, err
	}
	refreshEncrypted, err := m.encryptOptional(tokens.RefreshToken)
	if err != nil {
		return SessionIdentity{}, err
	}
	groups, _ := json.Marshal(claims.Groups)
	accessExpiresAt := tokenExpiry(time.Now(), tokens.ExpiresIn, claims.ExpiresAt)
	_, err = tx.Exec(ctx, `
		UPDATE configurator_auth_sessions SET username=$1,display_name=$2,groups_json=$3,platform_admin=$4,
			access_token_encrypted=$5,refresh_token_encrypted=$6,access_expires_at=$7,updated_at=NOW()
		WHERE token_hash=$8 AND revoked_at IS NULL`, claims.Username, claims.DisplayName, groups, claims.PlatformAdmin,
		accessEncrypted, nullableBytes(refreshEncrypted), accessExpiresAt, hashToken(cookieToken))
	if err != nil {
		return SessionIdentity{}, fmt.Errorf("update refreshed Configurator session: %w", err)
	}
	claims.ExpiresAt = accessExpiresAt
	return SessionIdentity{Claims: claims, SessionID: record.ID}, nil
}

func (m *SessionManager) safeReturnURL(candidate string) string {
	fallback, err := url.Parse(m.config.ReturnURL)
	if err != nil {
		return m.config.ReturnURL
	}
	requested, err := url.Parse(strings.TrimSpace(candidate))
	if err != nil || requested.String() == "" {
		return fallback.String()
	}
	if !requested.IsAbs() {
		requested = fallback.ResolveReference(requested)
	}
	if requested.Scheme != fallback.Scheme || !strings.EqualFold(requested.Host, fallback.Host) {
		return fallback.String()
	}
	return requested.String()
}

func (m *SessionManager) encrypt(value string) ([]byte, error) {
	if m.aead == nil {
		return nil, fmt.Errorf("Configurator session encryption is unavailable")
	}
	nonce := make([]byte, m.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}
	return m.aead.Seal(nonce, nonce, []byte(value), nil), nil
}

func (m *SessionManager) decrypt(value []byte) (string, error) {
	if m.aead == nil || len(value) < m.aead.NonceSize() {
		return "", fmt.Errorf("Configurator session encrypted value is invalid")
	}
	nonce := value[:m.aead.NonceSize()]
	plain, err := m.aead.Open(nil, nonce, value[m.aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt Configurator session value: %w", err)
	}
	return string(plain), nil
}

func (m *SessionManager) encryptOptional(value string) ([]byte, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return m.encrypt(value)
}

func (m *SessionManager) decryptOptional(value []byte) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	return m.decrypt(value)
}

type sessionRecord struct {
	ID                    string
	ProviderID            string
	Subject               string
	Issuer                string
	Username              string
	DisplayName           string
	Groups                []string
	PlatformAdmin         bool
	AccessTokenEncrypted  []byte
	RefreshTokenEncrypted []byte
	AccessExpiresAt       time.Time
	ExpiresAt             time.Time
}

func (r sessionRecord) identity() SessionIdentity {
	return SessionIdentity{Claims: Claims{ProviderID: r.ProviderID, Subject: r.Subject, Issuer: r.Issuer, Username: r.Username,
		DisplayName: r.DisplayName, Groups: r.Groups, PlatformAdmin: r.PlatformAdmin, ExpiresAt: r.AccessExpiresAt}, SessionID: r.ID}
}

func secureRandom(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate secure random value: %w", err)
	}
	return rawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func tokenExpiry(now time.Time, expiresIn int64, claimsExpiry time.Time) time.Time {
	if expiresIn > 0 {
		return now.Add(time.Duration(expiresIn) * time.Second)
	}
	return claimsExpiry
}
