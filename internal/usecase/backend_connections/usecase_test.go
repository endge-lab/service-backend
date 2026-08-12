package backend_connections

import (
	"context"
	"errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type repositoryStub struct {
	items   []entities.BackendConnection
	created *entities.BackendConnection
	deleted string
	err     error
}

func (r *repositoryStub) ListBackendConnections(context.Context) ([]entities.BackendConnection, error) {
	return r.items, r.err
}

func (r *repositoryStub) InsertBackendConnection(_ context.Context, value entities.BackendConnection) (*entities.BackendConnection, error) {
	r.created = &value
	return &value, r.err
}

func (r *repositoryStub) DeleteBackendConnection(_ context.Context, id string) error {
	r.deleted = id
	return r.err
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "trim and trailing slash", raw: "  https://Backend.Example.com/api///  ", want: "https://backend.example.com/api"},
		{name: "http", raw: "http://localhost:8080/", want: "http://localhost:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBaseURL(tt.raw)
			if err != nil || got != tt.want {
				t.Fatalf("NormalizeBaseURL(%q) = %q, %v; want %q", tt.raw, got, err, tt.want)
			}
		})
	}

	for _, raw := range []string{
		"backend.example.com",
		"ftp://backend.example.com",
		"https://user:password@backend.example.com",
		"https://backend.example.com?target=other",
		"https://backend.example.com?",
		"https://backend.example.com#fragment",
		"https://backend.example.com#",
	} {
		if _, err := NormalizeBaseURL(raw); err == nil {
			t.Fatalf("NormalizeBaseURL(%q) accepted invalid URL", raw)
		}
	}
}

func TestUseCaseRequiresPlatformAdminForMutations(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{}
	usecase := NewUseCase(repository)
	ctx := actorContext(false)

	if _, err := usecase.Create(ctx, "https://remote.example.com"); err == nil {
		t.Fatal("non-admin create was allowed")
	}
	if err := usecase.Delete(ctx, "550e8400-e29b-41d4-a716-446655440000"); err == nil {
		t.Fatal("non-admin delete was allowed")
	}
	if repository.created != nil || repository.deleted != "" {
		t.Fatal("repository was mutated before authorization")
	}
}

func TestUseCaseListAndMutations(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{items: []entities.BackendConnection{{ID: "connection", BaseURL: "https://remote.example.com"}}}
	usecase := NewUseCase(repository)

	result, err := usecase.List(actorContext(false))
	if err != nil || result.CanManage || len(result.Items) != 1 {
		t.Fatalf("unexpected viewer list result: %#v, %v", result, err)
	}
	created, err := usecase.Create(actorContext(true), " https://REMOTE.example.com/ ")
	if err != nil || created.BaseURL != "https://remote.example.com" {
		t.Fatalf("unexpected create result: %#v, %v", created, err)
	}

	repository.err = ports.ErrNotFound
	if err = usecase.Delete(actorContext(true), "550e8400-e29b-41d4-a716-446655440000"); err == nil {
		t.Fatal("missing id did not return an error")
	}
	repository.err = errors.New("duplicate key value violates unique constraint")
	if _, err = usecase.Create(actorContext(true), "https://remote.example.com"); err == nil {
		t.Fatal("duplicate URL did not return a conflict")
	}
}

func actorContext(platformAdmin bool) context.Context {
	return entities.WithCurrentActor(context.Background(), entities.CurrentActor{
		User:          &entities.User{ID: "550e8400-e29b-41d4-a716-446655440001"},
		PlatformAdmin: platformAdmin,
	})
}
