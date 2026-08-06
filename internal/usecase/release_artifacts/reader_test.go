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
}

func (s *artifactRepositoryStub) GetReleaseArtifact(ctx context.Context, _, _ string) (*entities.ReleaseArtifact, error) {
	s.mu.Lock()
	s.calls++
	artifact, err, delay, started, block := cloneArtifact(s.artifact), s.err, s.delay, s.started, s.block
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
	return &artifact, nil
}

func (s *artifactRepositoryStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestReaderCachesArtifactAndProtectsStoredBytes(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `{"version":1}`)}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	first, err := reader.Read(context.Background(), "export", release.WorkspaceID, release)
	if err != nil {
		t.Fatal(err)
	}
	first.Data[0] = '!'
	second, err := reader.Read(context.Background(), "export", release.WorkspaceID, release)
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

func TestReaderUsesWorkspaceChecksumAndLRU(t *testing.T) {
	repository := &artifactRepositoryStub{}
	reader := newReader(t, repository, enabledCache(4, 4))
	first := testRelease("workspace-a", "release-1", "first")
	second := testRelease("workspace-a", "release-2", "second")
	third := testRelease("workspace-a", "release-3", "third")

	for _, release := range []entities.Release{first, second, first, third} {
		repository.artifact = testArtifact(release, `[]`)
		if _, err := reader.Read(context.Background(), "export", release.WorkspaceID, release); err != nil {
			t.Fatal(err)
		}
	}
	if repository.callCount() != 3 {
		t.Fatalf("repository calls = %d, want 3", repository.callCount())
	}

	// second был самым давно использованным и должен быть вытеснен третьим.
	repository.artifact = testArtifact(second, `[]`)
	if _, err := reader.Read(context.Background(), "export", second.WorkspaceID, second); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 4 {
		t.Fatalf("LRU did not evict oldest value, calls = %d", repository.callCount())
	}

	otherWorkspace := testRelease("workspace-b", first.ID, first.Checksum)
	repository.artifact = testArtifact(otherWorkspace, `[]`)
	if _, err := reader.Read(context.Background(), "export", otherWorkspace.WorkspaceID, otherWorkspace); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 5 {
		t.Fatalf("workspaces shared one cache key, calls = %d", repository.callCount())
	}
}

func TestReaderDropsOldChecksumVersion(t *testing.T) {
	oldRelease := testRelease("workspace", "release", "old")
	newRelease := testRelease("workspace", "release", "new")
	repository := &artifactRepositoryStub{artifact: testArtifact(oldRelease, `{}`)}
	reader := newReader(t, repository, enabledCache(1024, 1024))

	if _, err := reader.Read(context.Background(), "export", oldRelease.WorkspaceID, oldRelease); err != nil {
		t.Fatal(err)
	}
	repository.artifact = testArtifact(newRelease, `[]`)
	if _, err := reader.Read(context.Background(), "export", newRelease.WorkspaceID, newRelease); err != nil {
		t.Fatal(err)
	}
	if len(reader.cache.byKey) != 1 {
		t.Fatalf("cached versions = %d, want 1", len(reader.cache.byKey))
	}
	if _, ok := reader.cache.byKey[artifactKey{workspaceID: "workspace", releaseID: "release", checksum: "new"}]; !ok {
		t.Fatal("new checksum was not cached")
	}
}

func TestReaderBypassesOversizedArtifactAndDoesNotCacheErrors(t *testing.T) {
	release := testRelease("workspace", "release", "checksum")
	repository := &artifactRepositoryStub{artifact: testArtifact(release, `123`)}
	reader := newReader(t, repository, enabledCache(10, 2))

	for range 2 {
		if _, err := reader.Read(context.Background(), "export", release.WorkspaceID, release); err != nil {
			t.Fatal(err)
		}
	}
	if repository.callCount() != 2 || len(reader.cache.byKey) != 0 {
		t.Fatalf("oversized artifact was cached: calls=%d items=%d", repository.callCount(), len(reader.cache.byKey))
	}

	failed := testRelease("workspace", "failed", "checksum")
	repository.err = errors.New("postgres unavailable")
	if _, err := reader.Read(context.Background(), "export", failed.WorkspaceID, failed); err == nil {
		t.Fatal("repository error was swallowed")
	}
	repository.err = nil
	repository.artifact = testArtifact(failed, `{}`)
	if _, err := reader.Read(context.Background(), "export", failed.WorkspaceID, failed); err != nil {
		t.Fatal(err)
	}
	if repository.callCount() != 4 {
		t.Fatalf("repository error was cached, calls = %d", repository.callCount())
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
	return entities.ReleaseArtifact{ReleaseID: release.ID, WorkspaceID: release.WorkspaceID, Checksum: release.Checksum, Data: []byte(data)}
}
