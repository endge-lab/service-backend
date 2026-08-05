//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/test/support"
)

// TestAuthSessionReadDoesNotAcquireRefreshLock проверяет, что обычная проверка
// актуальной browser session не ожидает блокировку строки, предназначенную для refresh.
func TestAuthSessionReadDoesNotAcquireRefreshLock(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	cfg := support.DevConfig()
	manager, err := auth.NewSessionManager(cfg, database.Pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	cookieToken := "unlocked-session"
	tokenHash := sha256.Sum256([]byte(cookieToken))
	_, err = database.Pool.Exec(t.Context(), `
		INSERT INTO configurator_auth_sessions(
			token_hash,provider_id,subject,issuer,username,display_name,groups_json,platform_admin,identity_refresh_at,expires_at)
		VALUES($1,'integration','unlocked-user','urn:endge:test','user','User','[]',FALSE,$2,$3)`,
		tokenHash[:], time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	locker, err := database.Pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = locker.Rollback(context.Background()) }()
	if _, err = locker.Exec(t.Context(), `SELECT id FROM configurator_auth_sessions WHERE token_hash=$1 FOR UPDATE`, tokenHash[:]); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	identity, err := manager.Resolve(ctx, cookieToken)
	if err != nil {
		t.Fatalf("чтение актуальной session заблокировалось на refresh lock: %v", err)
	}
	if identity.Subject != "unlocked-user" {
		t.Fatalf("получен неожиданный пользователь %q", identity.Subject)
	}
}

// TestAuthSessionCleanupRemovesOnlyExpiredState проверяет фоновую операцию
// очистки просроченных login transactions и browser sessions.
func TestAuthSessionCleanupRemovesOnlyExpiredState(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	manager, err := auth.NewSessionManager(support.DevConfig(), database.Pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Pool.Exec(t.Context(), `
		INSERT INTO configurator_auth_transactions(state_hash,browser_nonce_hash,verifier_encrypted,oidc_nonce_encrypted,return_url,expires_at)
		VALUES(decode(repeat('01',32),'hex'),decode(repeat('02',32),'hex'),'x','y','http://configurator.test',NOW()-INTERVAL '1 minute'),
		      (decode(repeat('03',32),'hex'),decode(repeat('04',32),'hex'),'x','y','http://configurator.test',NOW()+INTERVAL '1 hour')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Pool.Exec(t.Context(), `
		INSERT INTO configurator_auth_sessions(token_hash,provider_id,subject,issuer,identity_refresh_at,expires_at)
		VALUES(decode(repeat('05',32),'hex'),'integration','expired','urn:endge:test',NOW()-INTERVAL '2 hours',NOW()-INTERVAL '1 hour'),
		      (decode(repeat('06',32),'hex'),'integration','active','urn:endge:test',NOW()+INTERVAL '1 hour',NOW()+INTERVAL '2 hours')`)
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}

	var transactions, sessions int
	if err = database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM configurator_auth_transactions`).Scan(&transactions); err != nil {
		t.Fatal(err)
	}
	if err = database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM configurator_auth_sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if transactions != 1 || sessions != 1 {
		t.Fatalf("cleanup оставил transactions=%d sessions=%d, ожидалось по одной активной записи", transactions, sessions)
	}
}
