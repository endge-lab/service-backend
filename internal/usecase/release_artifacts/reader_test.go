package release_artifacts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"go.opentelemetry.io/otel/metric/noop"
)

type artifactRepositoryStub struct {
	mu        sync.Mutex
	calls     int
	artifact  entities.ReleaseArtifact
	err       error
	delay     time.Duration
	started   chan struct{}
	startOnce sync.Once
	block     <-chan struct{}
	returnNil bool
}

func (s *artifactRepositoryStub) GetReleaseArtifact(ctx context.Context, _, _ string) (*entities.ReleaseArtifact, error) {
	s.mu.Lock()
	s.calls++
	artifact, err, delay, started, block, returnNil := cloneArtifact(s.artifact), s.err, s.delay, s.started, s.block, s.returnNil
	s.mu.Unlock()
	if started != nil {
		s.startOnce.Do(func() { close(started) })
	}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	if err != nil {
		return nil, err
	}
	if returnNil {
		return nil, nil
	}
	return &artifact, nil
}

func (s *artifactRepositoryStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// TestReaderCachesArtifactAndProtectsStoredBytes читает один artifact дважды и
// проверяет cache hit, а также то, что изменение bytes у первого caller не портит LRU.
func TestReaderCachesArtifactAndProtectsStoredBytes(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `{"version":1}`)}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	first, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release)
	if err != nil {
		t.Fatal(err)
	}
	first.Data[0] = '!'
	second, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release)
	if err != nil {
		t.Fatal(err)
	}
	if string(second.Data) != `{"version":1}` {
		t.Fatalf("cached data was mutated: %s", second.Data)
	}
	if repository.callCount() != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.callCount())
	}
}

// TestReaderUsesWorkspaceChecksumAndLRU проверяет состав cache key и порядок LRU:
// недавний элемент сохраняется, старый вытесняется, а другой workspace не пересекается по key.
func TestReaderUsesWorkspaceChecksumAndLRU(t *testing.T) {
	repository := &artifactRepositoryStub{}
	reader := newReader(t, repository, enabledCache(4, 4))
	first := testRelease("workspace-a", "release-1", "first")
	second := testRelease("workspace-a", "release-2", "second")
	third := testRelease("workspace-a", "release-3", "third")

	for _, release := range []entities.Release{first, second, first, third} {
		repository.artifact = testArtifact(release, `[]`)
		if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release); err != nil {
			t.Fatal(err)
		}
	}
	if repository.callCount() != 3 {
		t.Fatalf("repository calls = %d, want 3", repository.callCount())
	}

	// second был самым давно использованным и должен быть вытеснен третьим.
	repository.artifact = testArtifact(second, `[]`)
	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, second.WorkspaceID, second); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 4 {
		t.Fatalf("LRU did not evict oldest value, calls = %d", repository.callCount())
	}

	otherWorkspace := testRelease("workspace-b", first.ID, first.Checksum)
	repository.artifact = testArtifact(otherWorkspace, `[]`)
	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, otherWorkspace.WorkspaceID, otherWorkspace); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 5 {
		t.Fatalf("workspaces shared one cache key, calls = %d", repository.callCount())
	}
}

// TestReaderDropsOldChecksumVersion проверяет защитный сценарий смены checksum:
// cache удаляет старые bytes того же release и оставляет только актуальную версию.
func TestReaderDropsOldChecksumVersion(t *testing.T) {
	oldRelease := testRelease("workspace", "release", "old")
	newRelease := testRelease("workspace", "release", "new")
	repository := &artifactRepositoryStub{artifact: testArtifact(oldRelease, `{}`)}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, oldRelease.WorkspaceID, oldRelease); err != nil {
		t.Fatal(err)
	}
	repository.artifact = testArtifact(newRelease, `[]`)
	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, newRelease.WorkspaceID, newRelease); err != nil {
		t.Fatal(err)
	}
	if len(reader.cache.byKey) != 1 {
		t.Fatalf("cached versions = %d, want 1", len(reader.cache.byKey))
	}
	if _, ok := reader.cache.byKey[artifactKey{workspaceID: "workspace", releaseID: "release", checksum: "new"}]; !ok {
		t.Fatal("new checksum was not cached")
	}
}

// TestReaderBypassesOversizedArtifactAndDoesNotCacheErrors проверяет два bypass-сценария:
// большой artifact и ошибка repository не должны стать cache hit при следующем Read.
func TestReaderBypassesOversizedArtifactAndDoesNotCacheErrors(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `123`)}
	reader := newReader(t, repository, enabledCache(10, 2))

	for range 2 {
		if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release); err != nil {
			t.Fatal(err)
		}
	}
	if repository.callCount() != 2 || len(reader.cache.byKey) != 0 {
		t.Fatalf("oversized artifact was cached: calls=%d items=%d", repository.callCount(), len(reader.cache.byKey))
	}

	failed := testRelease("workspace", "failed", "checksum")
	repository.err = errors.New("postgres unavailable")
	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, failed.WorkspaceID, failed); err == nil {
		t.Fatal("repository error was swallowed")
	}
	repository.err = nil
	repository.artifact = testArtifact(failed, `{}`)
	if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, failed.WorkspaceID, failed); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 4 {
		t.Fatalf("repository error was cached, calls = %d", repository.callCount())
	}
}

// TestReaderBypassesCacheWhenDisabled проверяет функциональный режим без LRU:
// artifact по-прежнему читается корректно, но каждый запрос идёт в repository.
func TestReaderBypassesCacheWhenDisabled(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `{"version":1}`)}
	reader := newReader(t, repository, config.ReleaseArtifactCacheConfig{Enabled: false})

	for range 2 {
		artifact, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release)
		if err != nil {
			t.Fatal(err)
		}
		if string(artifact.Data) != `{"version":1}` {
			t.Fatalf("artifact data = %s", artifact.Data)
		}
	}
	if repository.callCount() != 2 {
		t.Fatalf("repository calls = %d, want 2", repository.callCount())
	}
	reader.mu.Lock()
	cacheItems := len(reader.cache.byKey)
	reader.mu.Unlock()
	if cacheItems != 0 {
		t.Fatalf("cache items = %d, want 0", cacheItems)
	}
}

// TestReaderRejectsInconsistentArtifacts подставляет повреждённые repository results.
// Он проверяет, что Reader не кеширует их и после исправления repository повторяет загрузку.
func TestReaderRejectsInconsistentArtifacts(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	release.Identity = "production"
	valid := testArtifact(release, `{}`)

	tests := []struct {
		name   string
		mutate func(*artifactRepositoryStub)
	}{
		{name: "nil artifact", mutate: func(repository *artifactRepositoryStub) { repository.returnNil = true }},
		{name: "different release", mutate: func(repository *artifactRepositoryStub) { repository.artifact.ReleaseID = "other-release" }},
		{name: "different workspace", mutate: func(repository *artifactRepositoryStub) { repository.artifact.WorkspaceID = "other-workspace" }},
		{name: "different identity", mutate: func(repository *artifactRepositoryStub) { repository.artifact.Identity = "staging" }},
		{name: "different checksum", mutate: func(repository *artifactRepositoryStub) { repository.artifact.Checksum = "other-checksum" }},
		{name: "empty data", mutate: func(repository *artifactRepositoryStub) { repository.artifact.Data = nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &artifactRepositoryStub{artifact: valid}
			tt.mutate(repository)
			reader := newReader(t, repository, enabledCache(1024, 1024))

			if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release); err == nil {
				t.Fatal("Read() error = nil, want inconsistent artifact error")
			}
			reader.mu.Lock()
			cacheItems := len(reader.cache.byKey)
			reader.mu.Unlock()
			if cacheItems != 0 {
				t.Fatalf("cache items = %d, want 0", cacheItems)
			}

			repository.mu.Lock()
			repository.artifact = valid
			repository.returnNil = false
			repository.mu.Unlock()
			if _, err := reader.Read(context.Background(), ports.ReleaseArtifactOperationExport, release.WorkspaceID, release); err != nil {
				t.Fatalf("Read() after valid artifact: %v", err)
			}
			if repository.callCount() != 2 {
				t.Fatalf("repository calls = %d, want 2", repository.callCount())
			}
		})
	}
}

func newReader(t *testing.T, repository ports.ReleaseArtifactRepository, cache config.ReleaseArtifactCacheConfig) *Reader {
	t.Helper()
	reader, err := NewReader(repository, cache, noop.NewMeterProvider().Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func enabledCache(maxBytes, maxItemBytes int) config.ReleaseArtifactCacheConfig {
	return config.ReleaseArtifactCacheConfig{Enabled: true, MaxBytes: maxBytes, MaxItemBytes: maxItemBytes}
}

func testRelease(workspaceID, releaseID, checksum string) entities.Release {
	return entities.Release{ID: releaseID, WorkspaceID: workspaceID, Checksum: checksum}
}

func testArtifact(release entities.Release, data string) entities.ReleaseArtifact {
	return entities.ReleaseArtifact{ReleaseID: release.ID, WorkspaceID: release.WorkspaceID, Identity: release.Identity, Checksum: release.Checksum, Data: []byte(data)}
}
