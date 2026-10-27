package access_tokens

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type fakeWorkspaces struct {
	ports.WorkspaceRepository
	role string
}

func (f *fakeWorkspaces) GetWorkspace(_ context.Context, identity string) (*entities.Workspace, error) {
	if identity != "afl" {
		return nil, ports.ErrNotFound
	}
	return &entities.Workspace{ID: "ws-afl", Identity: "afl"}, nil
}

func (f *fakeWorkspaces) WorkspaceRole(context.Context, string, string, bool) (string, error) {
	return f.role, nil
}

type fakeTokens struct {
	created     []ports.CreateAccessTokenInput
	credential  *ports.AccessTokenCredential
	touched     int
	revokedBy   string
	listedOwner *string
}

func (f *fakeTokens) CreateAccessToken(_ context.Context, input ports.CreateAccessTokenInput) (*entities.AccessToken, error) {
	f.created = append(f.created, input)
	return &entities.AccessToken{ID: "token-1", Owner: entities.Actor{ID: input.UserID}, WorkspaceID: input.WorkspaceID, Scope: input.Scope,
		CanCommit: input.CanCommit, CanDelete: input.CanDelete, TokenPrefix: input.TokenPrefix, ExpiresAt: input.ExpiresAt}, nil
}

func (f *fakeTokens) ListAccessTokens(_ context.Context, userID *string) ([]entities.AccessToken, error) {
	f.listedOwner = userID
	return nil, nil
}

func (f *fakeTokens) GetAccessToken(_ context.Context, id string) (*entities.AccessToken, error) {
	if id != "token-1" {
		return nil, ports.ErrNotFound
	}
	return &entities.AccessToken{ID: id, Owner: entities.Actor{ID: "owner"}}, nil
}

func (f *fakeTokens) FindAccessTokenCredential(_ context.Context, hash []byte) (*ports.AccessTokenCredential, error) {
	if f.credential == nil || string(f.credential.SecretHash) != string(hash) {
		return nil, ports.ErrNotFound
	}
	copied := *f.credential
	return &copied, nil
}

func (f *fakeTokens) RevokeAccessToken(_ context.Context, _ string, actorID string) error {
	f.revokedBy = actorID
	return nil
}

func (f *fakeTokens) TouchAccessToken(context.Context, string, time.Duration) error {
	f.touched++
	return nil
}

var testNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func newTestUseCase(role string) (*UseCase, *fakeTokens) {
	tokens := &fakeTokens{}
	useCase := NewUseCase(tokens, &fakeWorkspaces{role: role}, 30*24*time.Hour)
	useCase.now = func() time.Time { return testNow }
	return useCase, tokens
}

func sessionContext(userID string, platformAdmin bool) context.Context {
	return entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: userID}, PlatformAdmin: platformAdmin})
}

func TestCreateIssuesReadTokenByDefaultAndStoresOnlyHash(t *testing.T) {
	useCase, tokens := newTestUseCase("editor")
	created, err := useCase.Create(sessionContext("owner", false), CreateInput{WorkspaceIdentity: "afl", Name: " Agent ", ExpiresAt: testNow.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(created.Secret, entities.AccessTokenSecretPrefix) || len(created.Secret) != len(entities.AccessTokenSecretPrefix)+encodedSecretChars {
		t.Fatal("unexpected secret format")
	}
	input := tokens.created[0]
	if input.Scope != entities.AccessTokenScopeRead || input.Name != "Agent" || input.WorkspaceID != "ws-afl" {
		t.Fatalf("unexpected input: %+v", input)
	}
	if string(input.SecretHash) != string(hashSecret(created.Secret)) || len(input.SecretHash) != 32 {
		t.Fatal("repository must receive SHA-256 of the secret")
	}
	if !strings.HasPrefix(created.Secret, input.TokenPrefix) || len(input.TokenPrefix) >= len(created.Secret) {
		t.Fatal("display prefix must be a strict prefix of the secret")
	}
}

func TestCreateValidation(t *testing.T) {
	cases := []struct {
		name  string
		role  string
		input CreateInput
		code  string
	}{
		{"expiry required", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a"}, "access_token_expiry_required"},
		{"expiry in past", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", ExpiresAt: testNow.Add(-time.Hour)}, "access_token_expiry_invalid"},
		{"expiry too long", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", ExpiresAt: testNow.Add(31 * 24 * time.Hour)}, "access_token_expiry_too_long"},
		{"flags need write", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", CanDelete: true, ExpiresAt: testNow.Add(time.Hour)}, "access_token_flags_require_write"},
		{"bad scope", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", Scope: "admin", ExpiresAt: testNow.Add(time.Hour)}, "access_token_scope_invalid"},
		{"empty name", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "  ", ExpiresAt: testNow.Add(time.Hour)}, "access_token_name_invalid"},
		{"description too long", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", Description: strings.Repeat("д", 1001), ExpiresAt: testNow.Add(time.Hour)}, "access_token_description_invalid"},
		{"unknown workspace", "editor", CreateInput{WorkspaceIdentity: "kzn", Name: "a", ExpiresAt: testNow.Add(time.Hour)}, "workspace_not_found"},
		{"no membership", "", CreateInput{WorkspaceIdentity: "afl", Name: "a", ExpiresAt: testNow.Add(time.Hour)}, "workspace_forbidden"},
		{"viewer cannot issue write", "viewer", CreateInput{WorkspaceIdentity: "afl", Name: "a", Scope: "write", ExpiresAt: testNow.Add(time.Hour)}, "access_token_scope_exceeds_role"},
		{"editor cannot issue commit", "editor", CreateInput{WorkspaceIdentity: "afl", Name: "a", Scope: "write", CanCommit: true, ExpiresAt: testNow.Add(time.Hour)}, "access_token_owner_admin_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useCase, _ := newTestUseCase(tc.role)
			_, err := useCase.Create(sessionContext("owner", false), tc.input)
			if domainerrors.CodeOf(err) != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestCreateCommitTokenForAdminOwner(t *testing.T) {
	for _, role := range []string{"admin", "platform_admin"} {
		useCase, tokens := newTestUseCase(role)
		input := CreateInput{WorkspaceIdentity: "afl", Name: "a", Scope: "write", CanCommit: true, ExpiresAt: testNow.Add(time.Hour)}
		if _, err := useCase.Create(sessionContext("owner", role == "platform_admin"), input); err != nil || !tokens.created[0].CanCommit {
			t.Fatalf("%s must issue a commit token: %v", role, err)
		}
	}
}

func TestTokenCannotManageTokens(t *testing.T) {
	useCase, _ := newTestUseCase("editor")
	ctx := entities.WithAccessToken(sessionContext("owner", true), entities.AccessTokenPrincipal{TokenID: "token-1", Scope: "write"})
	if _, err := useCase.Create(ctx, CreateInput{WorkspaceIdentity: "afl", Name: "a", ExpiresAt: testNow.Add(time.Hour)}); domainerrors.CodeOf(err) != "access_token_management_forbidden" {
		t.Fatalf("create via token: %v", err)
	}
	if _, err := useCase.List(ctx, false); domainerrors.CodeOf(err) != "access_token_management_forbidden" {
		t.Fatalf("list via token: %v", err)
	}
	if err := useCase.Revoke(ctx, "token-1"); domainerrors.CodeOf(err) != "access_token_management_forbidden" {
		t.Fatalf("revoke via token: %v", err)
	}
}

func TestListAndRevokeOwnership(t *testing.T) {
	useCase, tokens := newTestUseCase("editor")
	if _, err := useCase.List(sessionContext("owner", false), true); domainerrors.CodeOf(err) != "platform_admin_required" {
		t.Fatalf("all tokens without admin: %v", err)
	}
	if _, err := useCase.List(sessionContext("admin", true), true); err != nil || tokens.listedOwner != nil {
		t.Fatalf("admin list all: %v", err)
	}
	if err := useCase.Revoke(sessionContext("stranger", false), "token-1"); domainerrors.CodeOf(err) != "access_token_not_found" {
		t.Fatalf("foreign revoke must look like not found: %v", err)
	}
	if err := useCase.Revoke(sessionContext("admin", true), "token-1"); err != nil || tokens.revokedBy != "admin" {
		t.Fatalf("admin revoke: %v", err)
	}
}

func TestAuthenticateStatuses(t *testing.T) {
	useCase, tokens := newTestUseCase("editor")
	secret, prefix, hash, err := generateSecret()
	if err != nil {
		t.Fatal(err)
	}
	base := entities.AccessToken{ID: "token-1", WorkspaceID: "ws-afl", WorkspaceIdentity: "afl", Scope: "write", CanDelete: true, TokenPrefix: prefix, ExpiresAt: testNow.Add(time.Hour)}
	tokens.credential = &ports.AccessTokenCredential{Token: base, SecretHash: hash, Owner: entities.User{ID: "owner", Active: true}}

	result, err := useCase.Authenticate(context.Background(), secret)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if result.Principal.UserID != "owner" || result.Principal.WorkspaceIdentity != "afl" || !result.Principal.CanDelete || tokens.touched != 1 {
		t.Fatalf("unexpected principal %+v touched=%d", result.Principal, tokens.touched)
	}

	recent := testNow.Add(-30 * time.Second)
	tokens.credential.Token.LastUsedAt = &recent
	if _, err = useCase.Authenticate(context.Background(), secret); err != nil || tokens.touched != 1 {
		t.Fatalf("last_used_at must not be written more than once per minute: err=%v touched=%d", err, tokens.touched)
	}

	for _, tc := range []struct {
		name   string
		secret string
		mutate func(*entities.AccessToken)
		code   string
	}{
		{"malformed", "endge_pat_short", nil, "access_token_invalid"},
		{"jwt-like", "eyJhbGciOi", nil, "access_token_invalid"},
		{"unknown", entities.AccessTokenSecretPrefix + strings.Repeat("A", encodedSecretChars), nil, "access_token_invalid"},
		{"expired", secret, func(t *entities.AccessToken) { t.ExpiresAt = testNow }, "access_token_expired"},
		{"revoked", secret, func(t *entities.AccessToken) { t.RevokedAt = &testNow }, "access_token_revoked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tokens.credential.Token = base
			if tc.mutate != nil {
				tc.mutate(&tokens.credential.Token)
			}
			_, err := useCase.Authenticate(context.Background(), tc.secret)
			if domainerrors.CodeOf(err) != tc.code || domainerrors.HTTPStatusOf(err) != 401 {
				t.Fatalf("expected 401 %s, got %v", tc.code, err)
			}
		})
	}
}
