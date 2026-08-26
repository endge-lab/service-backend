package ai_catalog

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	platformencryption "github.com/endge-lab/service-backend/internal/platform/encryption"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type txStub struct{}

func (txStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (txStub) WithinReadTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type catalogStub struct {
	connections []entities.AIProviderConnection
	models      []entities.AIModelProfile
	cleared     bool
	deleted     string
	credential  []byte
}

func (r *catalogStub) ListAIProviderConnections(context.Context) ([]entities.AIProviderConnection, error) {
	return r.connections, nil
}
func (r *catalogStub) GetAIProviderConnection(_ context.Context, id string) (*ports.AIProviderConnectionRecord, error) {
	for _, value := range r.connections {
		if value.ID == id {
			return &ports.AIProviderConnectionRecord{Connection: value, Credential: r.credential}, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}
func (r *catalogStub) InsertAIProviderConnection(_ context.Context, value entities.AIProviderConnection, credential []byte) (*entities.AIProviderConnection, error) {
	r.connections = append(r.connections, value)
	r.credential = credential
	return &value, nil
}
func (r *catalogStub) UpdateAIProviderConnection(_ context.Context, value entities.AIProviderConnection) (*entities.AIProviderConnection, error) {
	return &value, nil
}
func (r *catalogStub) UpdateAIProviderCredential(_ context.Context, _ string, _ string, value []byte) (*entities.AIProviderConnection, error) {
	r.credential = value
	result := r.connections[0]
	result.HasCredential = true
	return &result, nil
}
func (r *catalogStub) DeleteAIProviderConnection(_ context.Context, id string) error {
	r.deleted = id
	return nil
}
func (r *catalogStub) ListAIModelProfiles(context.Context, bool) ([]entities.AIModelProfile, error) {
	return r.models, nil
}
func (r *catalogStub) GetAIModelProfile(_ context.Context, id string) (*entities.AIModelProfile, error) {
	for _, value := range r.models {
		if value.ID == id {
			copy := value
			return &copy, nil
		}
	}
	return nil, domainerrors.ErrNotFound
}
func (r *catalogStub) InsertAIModelProfile(_ context.Context, value entities.AIModelProfile) (*entities.AIModelProfile, error) {
	r.models = append(r.models, value)
	return &value, nil
}
func (r *catalogStub) UpdateAIModelProfile(_ context.Context, value entities.AIModelProfile) (*entities.AIModelProfile, error) {
	return &value, nil
}
func (r *catalogStub) ClearAIModelDefaults(context.Context, string) error {
	r.cleared = true
	return nil
}
func (r *catalogStub) DeleteAIModelProfile(_ context.Context, id string) error {
	r.deleted = id
	return nil
}

func TestCatalogStartsEmptyAndRequiresPlatformAdminForManagement(t *testing.T) {
	repo := &catalogStub{}
	usecase := newTestUseCase(t, repo)
	if _, err := usecase.ListConnections(actorContext(false)); domainerrors.HTTPStatusOf(err) != 403 {
		t.Fatalf("non platform admin status = %d, want 403", domainerrors.HTTPStatusOf(err))
	}
	connections, err := usecase.ListConnections(actorContext(true))
	if err != nil || len(connections) != 0 {
		t.Fatalf("empty catalog = %#v, %v", connections, err)
	}
}

func TestCreateConnectionEncryptsCredentialAndDeleteIsPhysical(t *testing.T) {
	repo := &catalogStub{}
	usecase := newTestUseCase(t, repo)
	created, err := usecase.CreateConnection(actorContext(true), "Local", "ollama", "http://localhost:11434", "secret", true)
	if err != nil {
		t.Fatal(err)
	}
	if string(repo.credential) == "secret" || len(repo.credential) == 0 || created.HasCredential {
		t.Fatalf("credential was not isolated: encrypted=%t response=%#v", len(repo.credential) > 0, created)
	}
	if err := usecase.DeleteConnection(actorContext(true), created.ID); err != nil || repo.deleted != created.ID {
		t.Fatalf("physical delete not delegated: id=%q err=%v", repo.deleted, err)
	}
}

func TestDefaultIsExplicitAndDeletingDoesNotSelectReplacement(t *testing.T) {
	connectionID := "d58d7f5f-b19e-440c-a090-2ecf39dd7298"
	repo := &catalogStub{connections: []entities.AIProviderConnection{{ID: connectionID, Enabled: true, Adapter: "ollama"}}}
	usecase := newTestUseCase(t, repo)
	first, err := usecase.CreateModel(actorContext(true), connectionID, "model-a", "Model A", true, false)
	if err != nil || first.Default || repo.cleared {
		t.Fatalf("first model became default: %#v cleared=%v err=%v", first, repo.cleared, err)
	}
	second, err := usecase.CreateModel(actorContext(true), connectionID, "model-b", "Model B", true, true)
	if err != nil || !second.Default || !repo.cleared {
		t.Fatalf("explicit default was not applied: %#v cleared=%v err=%v", second, repo.cleared, err)
	}
	repo.cleared = false
	if err := usecase.DeleteModel(actorContext(true), second.ID); err != nil || repo.cleared {
		t.Fatalf("delete selected replacement: cleared=%v err=%v", repo.cleared, err)
	}
}

func newTestUseCase(t *testing.T, repo *catalogStub) *UseCase {
	t.Helper()
	key := make([]byte, 32)
	for index := range key {
		key[index] = 7
	}
	keyring, err := platformencryption.NewKeyring(platformencryption.Config{Current: platformencryption.KeyConfig{ID: "v1", Key: base64.StdEncoding.EncodeToString(key)}})
	if err != nil {
		t.Fatal(err)
	}
	return NewUseCase(repo, txStub{}, keyring)
}

func actorContext(platformAdmin bool) context.Context {
	return entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: "59c995d6-61e8-45f1-ad09-baccdc9d62fc"}, PlatformAdmin: platformAdmin})
}
